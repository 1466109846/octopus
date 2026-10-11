package relay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/pipeline"
)

// TestUpstreamStatusOfExtractsFromErrorChain 状态码要能从两条转发路径的结构化错误链里提取出来。
//
// 保活页靠这个数字区分"上游响应了但拒绝"(429/503, 心跳已打到上游)与"根本连不上"(0),
// 任何一条路径漏提, 同一次失败在界面上就会显示成两种截然不同的结论。
//
// 提取只走结构化错误类型(见 upstreamStatusOf 的注释): 错误文本里随手出现的三位数字是正文,
// 不该被当成状态码。下面那组 "嵌入正文" 的用例就是守这道闸 —— 旧实现会扫文本找数字,
// 于是 "retries" 段落里的编号、地址里的端口都会污染状态码。
func TestUpstreamStatusOfExtractsFromErrorChain(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil 错误", nil, 0},
		{"透传: httpclient.Error 直接返回", &httpclient.Error{StatusCode: http.StatusTooManyRequests, Status: "429 Too Many Requests"}, 429},
		{"透传: httpclient.Error 被 fmt %w 包了一层(带正文)", fmt.Errorf("%w: %s", &httpclient.Error{StatusCode: http.StatusServiceUnavailable}, "queue full"), 503},
		{"转换: ResponseError 外包 UpstreamError", pipeline.WrapUpstreamError(&llm.ResponseError{StatusCode: http.StatusTooManyRequests}), 429},
		{"转换: 再被 %w 包一层(sendConverted 拼正文)", fmt.Errorf("%w: %s", pipeline.WrapUpstreamError(&llm.ResponseError{StatusCode: http.StatusBadGateway}), "body"), 502},
		{"流式首事件: 状态码走结构化错误而非文本前缀", pipeline.WrapUpstreamError(&llm.ResponseError{StatusCode: http.StatusTooManyRequests}), 429},
		// 下面是守"不解析嵌入正文"这一道闸。
		{"网络错误: 提取不到", errors.New("dial tcp 127.0.0.1:1: connect: connection refused"), 0},
		{"超时: 提取不到", errors.New("probe timeout after 45s"), 0},
		{"嵌入正文里的三位数不被当状态码", errors.New("rate limit reached after 429 retries, port 8080 busy"), 0},
		{"正文恰好以 503 开头也不是状态码", errors.New("503 queue full"), 0},
		{"旧式文本前缀不再被解析", errors.New("upstream responded 429 Too Many Requests: {\"error\":\"queue full\"}"), 0},
	}
	for _, item := range cases {
		if got := upstreamStatusOf(item.err); got != item.want {
			t.Fatalf("%s: expected %d, actual %d", item.name, item.want, got)
		}
	}
}

// TestUpstreamStatusOfRejectsOutOfRangeCodes 超出合法区间(100..599)的结构化状态码一律视为提取不到。
//
// 上游库即便抛出一个不合法的数字(0、负数、6xx), 也不该被当真: 0 与"无状态码"同义,
// 越界值不是真实 HTTP 状态码, 落到界面上只会误导"它回了什么"。
func TestUpstreamStatusOfRejectsOutOfRangeCodes(t *testing.T) {
	cases := []struct {
		name string
		code int
		want int
	}{
		{"0 不是真实状态码", 0, 0},
		{"99 低于下限", 99, 0},
		{"100 是下限", 100, 100},
		{"599 是上限", 599, 599},
		{"600 越界", 600, 0},
		{"999 越界", 999, 0},
		{"负数越界", -1, 0},
	}
	for _, tc := range cases {
		if got := upstreamStatusOf(&httpclient.Error{StatusCode: tc.code}); got != tc.want {
			t.Fatalf("%s: upstreamStatusOf(%d)=%d, want %d", tc.name, tc.code, got, tc.want)
		}
	}
}

// TestKeepaliveKindSemantics 保活任务永远不问糖果, 且被识别为保活; 监控任务保持原语义。
// 糖果题的回答再短也比最小请求大得多, 与保活"只花最少流量维持活跃"相悖, 所以由模型层统一关掉。
func TestKeepaliveKindSemantics(t *testing.T) {
	keepalive := model.ScheduledProbe{Kind: model.ScheduledProbeKindKeepalive, IQDisabled: false}
	if keepalive.IQProbeEnabled() {
		t.Fatal("保活任务即使 IQDisabled=false 也不该问糖果")
	}
	if !keepalive.IsKeepalive() {
		t.Fatal("kind=keepalive 应被识别为保活")
	}

	monitor := model.ScheduledProbe{Kind: model.ScheduledProbeKindMonitor, IQDisabled: false}
	if !monitor.IQProbeEnabled() || monitor.IsKeepalive() {
		t.Fatal("监控任务应照常问糖果, 且不是保活")
	}
	// 存量行: 升级前创建的任务读出来可能是空串, 必须按监控处理。
	legacy := model.ScheduledProbe{Kind: "", IQDisabled: false}
	if !legacy.IQProbeEnabled() || legacy.IsKeepalive() {
		t.Fatal("kind 为空的存量任务应按监控处理")
	}
}

// TestKeepaliveResultsIndependentFromMonitor 保活结论与监控结论各存一份, 互不覆盖、各自清理。
//
// 同一条凭据可以同时被两类任务覆盖: 共用一份表的话, 后落地的结论会盖掉先落地的,
// 两个页面上会看到同一条凭据显示着与各自任务无关的数字。
func TestKeepaliveResultsIndependentFromMonitor(t *testing.T) {
	keepaliveResultMu.Lock()
	keepaliveResults = make(map[int]ProbeResult)
	keepaliveResultMu.Unlock()
	scheduledProbeResultMu.Lock()
	scheduledProbeResults = make(map[int]ProbeResult)
	scheduledProbeResultMu.Unlock()

	now := time.Now().UnixMilli()
	recordScheduledProbeResult(77, ProbeResult{ItemID: 77, OK: true, StatusCode: 200, ProbedAt: now})
	recordKeepaliveResult(77, ProbeResult{ItemID: 77, OK: false, StatusCode: 429, Message: "queue full", ProbedAt: now})

	if got := ScheduledProbeResults([]int{77})[77]; !got.OK || got.StatusCode != 200 {
		t.Fatalf("监控结论被保活覆盖了: %+v", got)
	}
	if got := KeepaliveResults([]int{77})[77]; got.OK || got.StatusCode != 429 {
		t.Fatalf("保活结论被监控覆盖了: %+v", got)
	}

	ForgetKeepaliveResults([]int{77})
	if len(KeepaliveResults([]int{77})) != 0 {
		t.Fatal("ForgetKeepaliveResults 后保活结论应清空")
	}
	if len(ScheduledProbeResults([]int{77})) != 1 {
		t.Fatal("清理保活结论不应影响监控结论")
	}
}

// TestKeepaliveRetryableStatus 只有"上游忙/瞬时故障"值得重试。
//
// 边界的两侧各有代价: 把确定性拒绝(401/404/501)当忙态重试是白打三次上游;
// 把 0(超时/网络错误)当忙态重试是把一次挂起拖成三次 —— 两个方向都要守住。
func TestKeepaliveRetryableStatus(t *testing.T) {
	cases := []struct {
		code int
		want bool
	}{
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{0, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{418, false},
		{501, false},
		{505, false},
		{200, false},
	}
	for _, tc := range cases {
		if got := keepaliveRetryableStatus(tc.code); got != tc.want {
			t.Fatalf("keepaliveRetryableStatus(%d)=%v, want %v", tc.code, got, tc.want)
		}
	}
}

// TestKeepaliveWithRetrySucceedsAfterBusy 忙拒绝之后调通: 返回成功结论, 且确实敲了第二次门。
func TestKeepaliveWithRetrySucceedsAfterBusy(t *testing.T) {
	calls := 0
	result := keepaliveWithRetry(context.Background(), func() ProbeResult {
		calls++
		if calls == 1 {
			return ProbeResult{OK: false, StatusCode: 429, Message: "queue full"}
		}
		return ProbeResult{OK: true, StatusCode: 200}
	}, time.Millisecond, time.Minute)

	if !result.OK || result.StatusCode != 200 {
		t.Fatalf("should surface the successful retry: %+v", result)
	}
	if calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
	if result.Message != "" {
		t.Fatalf("success keeps message empty, got %q", result.Message)
	}
}

// TestKeepaliveWithRetryBoundsAttempts 一直忙: 敲满上限即收手, 结论带上重试次数。
//
// 上限存在的意义: 一轮保活不是无限敲门; 上游持续忙就让位给下一个周期,
// 而不把调度器(一拍只跑一条)占住。次数写进结论供界面区分"敲过几次都没开"。
func TestKeepaliveWithRetryBoundsAttempts(t *testing.T) {
	calls := 0
	result := keepaliveWithRetry(context.Background(), func() ProbeResult {
		calls++
		return ProbeResult{OK: false, StatusCode: 503, Message: "Service Unavailable"}
	}, time.Millisecond, time.Minute)

	if calls != keepaliveRetryAttempts {
		t.Fatalf("expected %d attempts, got %d", keepaliveRetryAttempts, calls)
	}
	if result.OK || result.StatusCode != 503 {
		t.Fatalf("should surface the last busy result: %+v", result)
	}
	if !strings.Contains(result.Message, "重试") || !strings.Contains(result.Message, "Service Unavailable") {
		t.Fatalf("message should keep upstream text and note retries: %q", result.Message)
	}
}

// TestKeepaliveWithRetryStopsOnDeterministicFailure 确定性拒绝(如 401)不重试: 一次即返回。
//
// 重试这类错误每一个字节都是白打上游 —— 钥匙不对不会因为隔了两秒就变对。
func TestKeepaliveWithRetryStopsOnDeterministicFailure(t *testing.T) {
	calls := 0
	result := keepaliveWithRetry(context.Background(), func() ProbeResult {
		calls++
		return ProbeResult{OK: false, StatusCode: 401, Message: "unauthorized"}
	}, time.Millisecond, time.Minute)

	if calls != 1 {
		t.Fatalf("deterministic failure must not be retried, got %d calls", calls)
	}
	if result.Message != "unauthorized" {
		t.Fatalf("message should be untouched: %q", result.Message)
	}
}

// TestKeepaliveWithRetryRespectsBudget 预算为负(等价于"没有等待额度")时首次忙拒绝即收手。
//
// 预算兜底的是"响应极慢的忙拒绝"这类畸形情形: 到点即收手, 不拖住调度节拍。
func TestKeepaliveWithRetryRespectsBudget(t *testing.T) {
	calls := 0
	result := keepaliveWithRetry(context.Background(), func() ProbeResult {
		calls++
		return ProbeResult{OK: false, StatusCode: 429}
	}, time.Second, 0)

	if calls != 1 {
		t.Fatalf("budget exhausted should prevent waiting: got %d calls", calls)
	}
	if result.StatusCode != 429 {
		t.Fatalf("should surface the first busy result: %+v", result)
	}
}

// TestKeepaliveWithRetryStopsOnContextCancel 等待期间上下文取消(如手动保活请求被客户端放弃): 不再敲门。
func TestKeepaliveWithRetryStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	result := keepaliveWithRetry(ctx, func() ProbeResult {
		calls++
		cancel()
		return ProbeResult{OK: false, StatusCode: 429}
	}, time.Second, time.Minute)

	if calls != 1 {
		t.Fatalf("canceled wait must not fire another probe, got %d calls", calls)
	}
	if result.StatusCode != 429 {
		t.Fatalf("should surface the first busy result: %+v", result)
	}
}

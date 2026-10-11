package relay

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
)

// 保活对内转/队列忙的健壮性: 排队类上游把"队列已满"回成 429/503 是常态, 而保活的目的
// 恰恰是"维持活跃、争取排队位"。收到一次忙拒绝就干等下一个周期(默认 5 分钟), 等于每轮只敲
// 一次门 —— 门没开就退到队尾, 与"持续排队"的诉求相反。
//
// 因此保活在同一轮内对"上游忙"做有界退避重试: 忙拒绝是上游已经作答的快速响应, 隔几秒再敲
// 一次, 队列一旦出现空位就能挤进去。三条边界, 每条都对应一类不该发生的事:
//   - 只重试"已达状态码的忙碌拒绝"(429/503): 超时与网络错误重试只会把挂起拖成三倍,
//     4xx 里除 429 外都是确定性拒绝(钥匙不对/模型不存在), 重试同样打不通;
//   - 最多 keepaliveRetryAttempts 次: 一轮保活不是无限敲门, 上游持续忙就让位给下一个周期;
//   - 整轮重试额外耗时不超过 keepaliveRetryBudget: 调度器一拍只跑一条探测(全局单飞),
//     慢速拒绝把整轮拖到分钟级会冻结其余任务的心跳。
const (
	// keepaliveRetryAttempts 是一轮保活里允许的总尝试次数(含首次)。
	// 取 3 而不是更多: 忙碌拒绝的恢复以秒计, 三次敲门覆盖约十秒的窗口;
	// 更长的等待属于"下一个保活周期"的语义, 不该让本条把调度器占住。
	keepaliveRetryAttempts = 3
	// keepaliveRetryBaseDelay 是首次重试前的等待, 之后逐次翻倍 (2s, 4s)。
	// 固定梯子而不是随机抖动: 单条保活是串行的, 不存在同刻自撞; 可预期的节奏也便于排查。
	keepaliveRetryBaseDelay = 2 * time.Second
	// keepaliveRetryBudget 是一轮保活允许为重试额外花掉的时间上限。
	// 正常的忙碌拒绝在毫秒级返回, 三次尝试加两次等待不过约 6 秒;
	// 预算兜底的是"响应极慢的 429/503"这类畸形情形, 到点即收手, 不拖住调度节拍。
	keepaliveRetryBudget = 60 * time.Second
)

// keepaliveResults 是各条渠道凭据最近一次保活探测的结论, 按授权主键索引。
//
// 与 scheduledProbeResults(监控结论)分开存储而不是共用一份:
// 同一条凭据可以同时被监控任务与保活任务覆盖, 两份结论回答的是不同的问题 ——
// 监控那份是"通道健不健康"(落路由、可冷却), 保活这份是"最近一次心跳打没打通、上游回了什么状态"(只展示)。
// 共用一份的话, 后落地的结论会盖掉先落地的, 两个页面上会看到同一条凭据显示着与各自任务无关的数字。
//
// 只存进程内, 不落库: 与监控结论同一口径, 重启后本来就该全部作废。
var (
	keepaliveResultMu sync.Mutex
	keepaliveResults  = make(map[int]ProbeResult)
)

// recordKeepaliveResult 记下一条凭据的保活结论, 新的直接覆盖旧的。
func recordKeepaliveResult(grantID int, result ProbeResult) {
	keepaliveResultMu.Lock()
	defer keepaliveResultMu.Unlock()

	keepaliveResults[grantID] = result
}

// KeepaliveResults 返回给定授权的保活结论, 按授权主键索引。
// 与监控结论同样不设有效期: 界面照实显示"多久以前", 该不该采信由看的人判断。
func KeepaliveResults(grantIDs []int) map[int]ProbeResult {
	keepaliveResultMu.Lock()
	defer keepaliveResultMu.Unlock()

	results := make(map[int]ProbeResult, len(grantIDs))
	for _, grantID := range grantIDs {
		if result, ok := keepaliveResults[grantID]; ok {
			results[grantID] = result
		}
	}
	return results
}

// ForgetKeepaliveResults 丢掉这些授权上的保活结论, 供任务被删掉时调用, 语义与 ForgetScheduledProbeResults 一致。
func ForgetKeepaliveResults(grantIDs []int) {
	keepaliveResultMu.Lock()
	defer keepaliveResultMu.Unlock()

	for _, grantID := range grantIDs {
		delete(keepaliveResults, grantID)
	}
}

// ProbeKeepaliveGrant 对一条渠道授权做一次保活探测: 发最小请求, 结论只落保活结果表。
//
// 与 ProbeScheduledGrant(监控)的关键差别就在落点: 监控要把结论落进分组路由状态(失败即冷却、参与选路),
// 保活**一律不落路由** —— 排队类上游把"队列已满"回成 429/503 是常态, 照监控语义冷却渠道,
// 等于把一条完全正常的渠道越保越死, 与"维持活跃"的目的正好相反。
// 同一份结论也不写监控结果表: 两个页面各看各的, 互不污染(见 keepaliveResults 的说明)。
//
// 复用 ProbeChannelGrant 而不是另写一套请求: 那正是模型页一次性测活走的路径 ——
// 取授权、构造请求、走真实的透传/转换发送逻辑, 且本来就不落路由状态, 语义完全对得上。
//
// 对"上游忙"的拒绝在同一轮内退避重试, 见 keepaliveWithRetry 与文件头说明:
// 保活要的就是排队位, 敲一次门没开就干等下一个周期与"持续排队"的诉求相反。
func ProbeKeepaliveGrant(ctx context.Context, grantID int) ProbeResult {
	result := keepaliveWithRetry(ctx, func() ProbeResult {
		return ProbeChannelGrant(ctx, grantID, false)
	}, keepaliveRetryBaseDelay, keepaliveRetryBudget)
	recordKeepaliveResult(grantID, result)
	return result
}

// keepaliveWithRetry 以有界退避重试执行 probe: 首次结果不是"上游忙"就直接返回; 是忙拒绝则隔一段
// 再敲一次, 直到成功、遇到非忙错误、用满次数或触到预算 —— 四条出口都返回最后一次真实探测的结论。
//
// probe 与两个时长都做成参数: 重试策略要能在测试里被直接驱动(不必打真实上游、不必真等秒级间隔),
// 生产调用点只传常量。
func keepaliveWithRetry(ctx context.Context, probe func() ProbeResult, baseDelay, budget time.Duration) ProbeResult {
	startedAt := time.Now()
	attempts := 1
	result := probe()
	for keepaliveRetryableStatus(result.StatusCode) && attempts < keepaliveRetryAttempts {
		// 预算从本轮开始时算起, 且在等待发生之前判: 到点即收手, 不让"敲门"把调度节拍拖住。
		if delay := baseDelay << (attempts - 1); time.Since(startedAt)+delay <= budget {
			select {
			case <-ctx.Done():
				return result
			case <-time.After(delay):
			}
		} else {
			break
		}
		attempts++
		result = probe()
	}
	if attempts > 1 && !result.OK {
		// 结论里带上重试次数: 界面能区分"敲过一次门没开"与"敲了三次都没开", 也就知道这套健壮性真的在跑。
		if result.Message == "" {
			result.Message = fmt.Sprintf("重试 %d 次仍被拒", attempts-1)
		} else {
			result.Message = fmt.Sprintf("%s（重试 %d 次仍被拒）", result.Message, attempts-1)
		}
	}
	return result
}

// keepaliveRetryableStatus 判断一次保活失败是否值得在同一轮内重试: 只认"上游忙或瞬时服务端错误"。
//
//   - 429: 限流/队列满, 排队类上游的常态拒绝 —— 隔几秒再敲门正好对应"排队";
//   - 500/502/503/504: 上游内部错误与服务不可用; anyrouter 类上游把模型负载上限也回成 500
//     (code=get_channel_failed, 正文本就是"请稍后重试"), 属同类忙态;
//   - 不重试 0(超时/网络错误): 重试只会把一次挂起拖成三次, 且几秒的等待大多等不来恢复;
//   - 不重试除 429 外的 4xx(钥匙不对/模型不存在等确定性拒绝), 501/505(未实现/版本不支持)同样排除 ——
//     它们是"没实现", 不是"忙"。
func keepaliveRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// ProbeKeepaliveNow 手动把一条保活任务的每条凭据都保活一遍。
// 与 ProbeScheduledNow 同一条并发约束(同渠道串行、跨渠道最多 probeConcurrency 路并发):
// 保活打的同样是真实上游, 一次全部并发会留下一轮突发流量。
//
// 只接保活任务: 拿监控任务的 ID 进来会被拒绝 —— 那会让监控的凭据行上出现一条与任务无关的
// 保活结论, 且删掉监控任务时按种类清理不到这份结论。
func ProbeKeepaliveNow(ctx context.Context, probe model.ScheduledProbe) ([]ProbeResult, error) {
	if err := guardProbeKindFor(probe, model.ScheduledProbeKindKeepalive); err != nil {
		return nil, err
	}
	credits := op.ScheduledProbeCredits(probe)
	if len(credits) == 0 {
		return nil, fmt.Errorf("no available credential for this keepalive")
	}
	channelIDs := make([]int, len(credits))
	for index, credit := range credits {
		channelIDs[index] = credit.Target.ChannelID
	}
	return probeByChannel(channelIDs, func(index int) ProbeResult {
		return ProbeKeepaliveGrant(ctx, credits[index].GrantID)
	}), nil
}

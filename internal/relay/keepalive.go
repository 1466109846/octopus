package relay

import (
	"context"
	"fmt"
	"sync"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
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
func ProbeKeepaliveGrant(ctx context.Context, grantID int) ProbeResult {
	result := ProbeChannelGrant(ctx, grantID, false)
	recordKeepaliveResult(grantID, result)
	return result
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

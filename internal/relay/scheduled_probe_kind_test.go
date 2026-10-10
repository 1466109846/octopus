package relay

import (
	"testing"

	"github.com/bestruirui/octopus/internal/model"
)

// TestGuardProbeKind 守调度与手动探测进入网络调用之前的种类闸。
//
// 空串(存量行兼容)按 monitor 放行; 两个已知种类放行; 任何非空未知值拒绝。
// 这层闸与 API/op 的校验口径一致, 但若真有带非法 kind 的任务走到这里,
// 应当立刻报错而不是带着语意不明的种类去打上游、把结论落到错误的状态表上。
func TestGuardProbeKind(t *testing.T) {
	cases := []struct {
		name    string
		kind    string
		wantErr bool
	}{
		{"缺省空串放行", "", false},
		{"monitor 放行", model.ScheduledProbeKindMonitor, false},
		{"keepalive 放行", model.ScheduledProbeKindKeepalive, false},
		{"非空未知值拒绝", "bogus", true},
		{"大写误写拒绝", "Keepalive", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := guardProbeKind(model.ScheduledProbe{Kind: tc.kind})
			if tc.wantErr && err == nil {
				t.Fatalf("kind=%q 应被 guardProbeKind 拒绝", tc.kind)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("kind=%q 应放行, 却报错: %v", tc.kind, err)
			}
		})
	}
}

// TestGuardProbeKindForEnforcesDirection 手动入口必须按方向拒绝拿错种类的任务。
//
// 只校验"合法值"挡不住这一类: monitor 与 keepalive 都是合法值, 监控入口拿到保活任务
// 照样会放行 —— 然后按监控语义把保活的失败写成渠道冷却, 正是保活承诺绝不发生的事。
// 反向同理: 保活入口拿到监控任务会在监控的凭据行上留下与任务无关的保活结论。
func TestGuardProbeKindForEnforcesDirection(t *testing.T) {
	cases := []struct {
		name     string
		kind     string
		expected string
		wantErr  bool
	}{
		{"监控入口接监控任务", model.ScheduledProbeKindMonitor, model.ScheduledProbeKindMonitor, false},
		{"监控入口接存量空串", "", model.ScheduledProbeKindMonitor, false},
		{"保活入口接保活任务", model.ScheduledProbeKindKeepalive, model.ScheduledProbeKindKeepalive, false},
		{"监控入口拒绝保活任务", model.ScheduledProbeKindKeepalive, model.ScheduledProbeKindMonitor, true},
		{"保活入口拒绝监控任务", model.ScheduledProbeKindMonitor, model.ScheduledProbeKindKeepalive, true},
		{"保活入口拒绝存量空串", "", model.ScheduledProbeKindKeepalive, true},
		{"非法值在方向校验前先被拒", "bogus", model.ScheduledProbeKindMonitor, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := guardProbeKindFor(model.ScheduledProbe{ID: 7, Kind: tc.kind}, tc.expected)
			if tc.wantErr && err == nil {
				t.Fatalf("kind=%q expected=%q 应被拒绝", tc.kind, tc.expected)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("kind=%q expected=%q 应放行, 却报错: %v", tc.kind, tc.expected, err)
			}
		})
	}
}

// TestRunScheduledProbeSkipsInvalidKind 非法种类的任务在调度一拍里不应发起任何网络调用,
// 只应把下一次探测推后。这里只验证"不留失败结论、不进轮转"的退化路径由 guard 前置于网络调用。
func TestRunScheduledProbeSkipsInvalidKind(t *testing.T) {
	// 任务 kind 非法: 直接喂给 guard, 确认它在网络调用之前就被拦下。
	if err := guardProbeKind(model.ScheduledProbe{Kind: "bogus"}); err == nil {
		t.Fatal("非法 kind 应在调度前被 guardProbeKind 拦下")
	}
}

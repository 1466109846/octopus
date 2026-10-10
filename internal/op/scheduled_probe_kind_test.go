package op

import (
	"testing"

	"github.com/bestruirui/octopus/internal/model"
)

// TestNormalizeProbeKind 守落库侧的种类折算: keepalive 直通, 空串与缺省按 monitor 兜底。
//
// 非空未知值不在这里出现: 提交校验(validateScheduledProbe → model.ValidateProbeKind)已经先行拒绝,
// 归一化只负责"空串兜底 + keepalive 直通", 不再把乱写的值静默回落成 monitor。
func TestNormalizeProbeKind(t *testing.T) {
	cases := []struct {
		name string
		kind string
		want string
	}{
		{"缺省空串回落 monitor", "", model.ScheduledProbeKindMonitor},
		{"monitor 直通", model.ScheduledProbeKindMonitor, model.ScheduledProbeKindMonitor},
		{"keepalive 直通", model.ScheduledProbeKindKeepalive, model.ScheduledProbeKindKeepalive},
	}
	for _, tc := range cases {
		if got := normalizeProbeKind(tc.kind); got != tc.want {
			t.Fatalf("%s: normalizeProbeKind(%q)=%q, want %q", tc.name, tc.kind, got, tc.want)
		}
	}
}

// TestValidateScheduledProbeRejectsUnknownKind 提交校验必须拒绝非空未知种类。
//
// 这是把"未知种类静默回落成 monitor"这条旧行为拦下来的最后一道闸: 一旦放行,
// 一条本意是保活的任务会被静默改成监控, 结论落点与调度语义全错且无任何报错。
// 空串(缺省/存量兼容)与两个已知种类仍应放行。
//
// 这里只播一个最小渠道配置把校验推过"种类之后"的环节, 让断言盯的确实是种类这一道,
// 而不是被"渠道不存在"之类的配置错误冲掉。
func TestValidateScheduledProbeRejectsUnknownKind(t *testing.T) {
	channelCache.Set(2, model.Channel{ID: 2, ChannelConfig: model.ChannelConfig{Name: "Demo", Enabled: true}})
	channelModelCache.Set(20, model.ChannelModel{ID: 20, ChannelID: 2, Name: "m-a"})
	t.Cleanup(func() {
		channelCache.Clear()
		channelModelCache.Clear()
	})

	validTarget := model.ScheduledProbeTargetRequest{ChannelID: 2, ModelName: "m-a"}

	if err := validateScheduledProbe(model.ScheduledProbeRequest{
		Name:    "ok",
		Targets: []model.ScheduledProbeTargetRequest{validTarget},
		Kind:    "bogus",
	}); err == nil {
		t.Fatal("非空未知种类应被 validateScheduledProbe 拒绝")
	}

	for _, kind := range []string{"", model.ScheduledProbeKindMonitor, model.ScheduledProbeKindKeepalive} {
		if err := validateScheduledProbe(model.ScheduledProbeRequest{
			Name:    "ok",
			Targets: []model.ScheduledProbeTargetRequest{validTarget},
			Kind:    kind,
		}); err != nil {
			t.Fatalf("kind=%q 应放行, 却报错: %v", kind, err)
		}
	}
}

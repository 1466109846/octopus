package relay

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm/httpclient"
)

// TestApplyClientEmulationClaudeCodeHeaders Claude Code 伪装要补齐这组身份头。
//
// User-Agent 与 X-App 是上游按客户端身份放行的凭据; anthropic-beta 的 1M 头是实测
// 硬需求 —— anyrouter 类中转在缺它时以"请启用 1m 上下文后重试"直接拒绝,
// 这条断言守住"默认带上"这个已确认的决策。
func TestApplyClientEmulationClaudeCodeHeaders(t *testing.T) {
	headers := http.Header{}
	if err := ApplyClientEmulation(model.ClientEmulationClaudeCode, headers); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"User-Agent":        claudeCodeUserAgent,
		"X-App":             "cli",
		"Anthropic-Version": claudeCodeVersion,
		"Anthropic-Beta":    claudeCodeContext1MBeta,
	}
	for key, value := range want {
		if got := headers.Get(key); got != value {
			t.Fatalf("%s: expected %q, actual %q", key, value, got)
		}
	}
}

// TestApplyClientEmulationClaudeCodePreservesExisting 已有头不被覆盖, beta 头按叠加处理。
//
// 两个方向:
//   - 身份头 (User-Agent/X-App/Anthropic-Version): 调用方 (客户端透传或转换器) 已给定就尊重它,
//     伪装只补空缺 —— 覆盖会让"想用自己的身份"的渠道失去表达能力。
//   - beta 头: 列表语义。已有其他 beta (如 interleaved-thinking) 时追加 1M 而不顶掉;
//     已含 1M 时不重复追加 (上游可能对重复 beta 报 400)。
func TestApplyClientEmulationClaudeCodePreservesExisting(t *testing.T) {
	headers := http.Header{
		"User-Agent":        []string{"custom-agent/9.9"},
		"X-App":             []string{"desktop"},
		"Anthropic-Version": []string{"2024-01-01"},
		"Anthropic-Beta":    []string{"interleaved-thinking-2025-05-14"},
	}
	if err := ApplyClientEmulation(model.ClientEmulationClaudeCode, headers); err != nil {
		t.Fatal(err)
	}
	if got := headers.Get("User-Agent"); got != "custom-agent/9.9" {
		t.Fatalf("User-Agent was overwritten: %q", got)
	}
	if got := headers.Get("X-App"); got != "desktop" {
		t.Fatalf("X-App was overwritten: %q", got)
	}
	if got := headers.Get("Anthropic-Version"); got != "2024-01-01" {
		t.Fatalf("Anthropic-Version was overwritten: %q", got)
	}
	if got := headers.Get("Anthropic-Beta"); got != "interleaved-thinking-2025-05-14,"+claudeCodeContext1MBeta {
		t.Fatalf("beta headers not appended: %q", got)
	}

	// 已含 1M 时不得重复追加。
	headers = http.Header{"Anthropic-Beta": []string{claudeCodeContext1MBeta}}
	if err := ApplyClientEmulation(model.ClientEmulationClaudeCode, headers); err != nil {
		t.Fatal(err)
	}
	if got := headers.Get("Anthropic-Beta"); got != claudeCodeContext1MBeta {
		t.Fatalf("beta token duplicated: %q", got)
	}
}

// TestApplyClientEmulationCodexHeaders Codex 伪装要补齐 originator / OpenAI-Beta / 会话头组。
//
// new-api 系中转的 codex 通道按 originator=codex_cli_rs 与 OpenAI-Beta: responses=experimental
// 判定请求来自官方 Codex CLI, 缺任一项即返回 invalid codex request; 会话头组是让请求
// 形状完整的陪衬, 上游只做归组。这里逐项断言, 防止将来重构时悄悄丢项。
func TestApplyClientEmulationCodexHeaders(t *testing.T) {
	headers := http.Header{}
	if err := ApplyClientEmulation(model.ClientEmulationCodex, headers); err != nil {
		t.Fatal(err)
	}
	if got := headers.Get("Originator"); got != codexOriginator {
		t.Fatalf("Originator: %q", got)
	}
	if got := headers.Get("User-Agent"); got != codexOriginator+"/"+codexVersion {
		t.Fatalf("User-Agent: %q", got)
	}
	if got := headers.Get("Version"); got != codexVersion {
		t.Fatalf("Version: %q", got)
	}
	if got := headers.Get("OpenAI-Beta"); got != codexBeta {
		t.Fatalf("OpenAI-Beta: %q", got)
	}

	// 会话头以同一个 session ID 贯通, window 带 :0 下标; 形状与 Codex CLI 一致。
	session := headers.Get("Session-Id")
	if session == "" {
		t.Fatal("Session-Id missing")
	}
	if got := headers.Get("Thread-Id"); got != session {
		t.Fatalf("Thread-Id not aligned with session: %q", got)
	}
	if got := headers.Get("Conversation_id"); got != session {
		t.Fatalf("Conversation_id not aligned with session: %q", got)
	}
	if got := headers.Get("X-Codex-Window-Id"); got != session+":0" {
		t.Fatalf("X-Codex-Window-Id: %q", got)
	}
	if got := headers.Get("X-Client-Request-Id"); got != session {
		t.Fatalf("X-Client-Request-Id: %q", got)
	}

	// Turn 元数据必须是可解析 JSON 且 session 对齐, 上游按它归组对话轮次。
	var turn map[string]any
	if err := json.Unmarshal([]byte(headers.Get("X-Codex-Turn-Metadata")), &turn); err != nil {
		t.Fatalf("turn metadata not JSON: %v", err)
	}
	if turn["session_id"] != session || turn["request_kind"] != "turn" {
		t.Fatalf("turn metadata shape drift: %v", turn)
	}
}

// TestApplyClientEmulationNoneAndUnknown 不伪装时一个头都不加; 未知模式按错误返回而不是静默忽略。
//
// 静默忽略未知模式等于"用户以为伪装生效、实际裸奔", 报错才能让渠道配置问题在保存/转发时暴露。
func TestApplyClientEmulationNoneAndUnknown(t *testing.T) {
	headers := http.Header{}
	if err := ApplyClientEmulation(model.ClientEmulationNone, headers); err != nil {
		t.Fatal(err)
	}
	if len(headers) != 0 {
		t.Fatalf("none emulation added headers: %v", headers)
	}

	err := ApplyClientEmulation(model.ClientEmulation("unknown_mode"), headers)
	if err == nil || !strings.Contains(err.Error(), "unsupported client emulation") {
		t.Fatalf("unknown emulation should error, got: %v", err)
	}
}

// TestApplyChannelConfigCustomHeaderOverridesEmulation 顺序契约: 伪装先铺底, custom_header 最后覆盖。
//
// 这是"伪装默认值可被渠道显式覆盖"的表达能力所在 —— e.g. 上游要求别的 UA 或用户想改用
// 自己的 beta 组合。整条链路走 applyChannelConfig, 保证转发与测活看到的顺序一致。
func TestApplyChannelConfigCustomHeaderOverridesEmulation(t *testing.T) {
	request := &httpclient.Request{Headers: http.Header{}}
	channel := model.Channel{ChannelConfig: model.ChannelConfig{
		ClientEmulation: model.ClientEmulationClaudeCode,
		CustomHeader: []model.CustomHeader{
			{HeaderKey: "User-Agent", HeaderValue: "override-agent/1.0"},
			{HeaderKey: "Anthropic-Beta", HeaderValue: "custom-beta-token"},
		},
	}}
	if err := applyChannelConfig(channel, request); err != nil {
		t.Fatal(err)
	}
	if got := request.Headers.Get("User-Agent"); got != "override-agent/1.0" {
		t.Fatalf("custom header did not win: %q", got)
	}
	if got := request.Headers.Get("Anthropic-Beta"); got != "custom-beta-token" {
		t.Fatalf("custom beta did not win: %q", got)
	}
	// 未被自定义覆盖的伪装头仍在。
	if got := request.Headers.Get("X-App"); got != "cli" {
		t.Fatalf("untouched emulation header missing: %q", got)
	}
}

// TestApplyChannelConfigEmulationDisabledByDefault 默认渠道 (无伪装配置) 不产生任何身份头。
//
// 伪装是 opt-in: 不开的渠道行为必须与旧版逐字节一致, 否则上线即全量改变所有渠道的出站形状。
func TestApplyChannelConfigEmulationDisabledByDefault(t *testing.T) {
	request := &httpclient.Request{Headers: http.Header{}}
	if err := applyChannelConfig(model.Channel{}, request); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"User-Agent", "X-App", "Anthropic-Beta", "Originator", "OpenAI-Beta"} {
		if got := request.Headers.Get(key); got != "" {
			t.Fatalf("default channel should not set %s, got %q", key, got)
		}
	}
}

package relay

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/tidwall/gjson"
)

// claudeCodeMessageRequest 造一个走 Messages 端点的伪装请求, 内容由调用方给。
func claudeCodeMessageRequest(body string) *httpclient.Request {
	return &httpclient.Request{
		APIFormat: string(llm.APIFormatAnthropicMessage),
		Body:      []byte(body),
	}
}

// TestApplyClientEmulationBodyAddsIdentityShape 无 system / 无 metadata 时补齐 CC 身份形状。
//
// 实测硬门槛 (anyrouter fable 类模型): system 首块必须是官方标记行, metadata.user_id 必须是
// 可解析的 v2 JSON (device_id + 合法 UUID 的 session_id)。缺任一项时轻体请求恒定 503,
// 而补齐后同分钟可拿到 200 真实回复 —— 这两项是"测活/转发能看到真实上游"的前提。
func TestApplyClientEmulationBodyAddsIdentityShape(t *testing.T) {
	request := claudeCodeMessageRequest(`{"model":"m","max_tokens":16,"stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	if err := ApplyClientEmulation(model.ClientEmulationClaudeCode, request.Headers); err != nil {
		t.Fatal(err)
	}
	if err := applyClientEmulationBody(model.ClientEmulationClaudeCode, request); err != nil {
		t.Fatal(err)
	}

	system := gjson.GetBytes(request.Body, "system")
	if !system.IsArray() || len(system.Array()) != 1 {
		t.Fatalf("system shape: %s", system.Raw)
	}
	if got := system.Array()[0].Get("text").String(); got != claudeCodeIdentityLine {
		t.Fatalf("identity line: %q", got)
	}
	if got := system.Array()[0].Get("type").String(); got != "text" {
		t.Fatalf("identity block type: %q", got)
	}

	userID := gjson.GetBytes(request.Body, "metadata.user_id").String()
	if !claudeCodeUserIDValid(userID) {
		t.Fatalf("generated user_id not valid: %q", userID)
	}
	var parsed struct {
		DeviceID  string `json:"device_id"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(userID), &parsed); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(parsed.DeviceID) {
		t.Fatalf("device_id not 64-hex: %q", parsed.DeviceID)
	}
	if !claudeCodeUUIDPattern.MatchString(parsed.SessionID) {
		t.Fatalf("session_id not uuid-shaped: %q", parsed.SessionID)
	}
}

// TestApplyClientEmulationBodyPreservesValidShape 已有合法形状逐字节保留。
//
// 真实 Claude Code 客户端经本网关转发时 (透传或转换), 请求体已经是对的;
// 伪装层只补缺, 不能改写 —— 否则"转发真实客户端"的路径反而被网关改坏。
func TestApplyClientEmulationBodyPreservesValidShape(t *testing.T) {
	body := `{"model":"m","system":[{"type":"text","text":"` + claudeCodeIdentityLine + `"},{"type":"text","text":"project context"}],` +
		`"metadata":{"user_id":"{\"device_id\":\"` + strings.Repeat("a", 64) + `\",\"account_uuid\":\"\",\"session_id\":\"61ed3771-7779-4014-8249-b39daeebd88c\"}"},` +
		`"messages":[{"role":"user","content":"hi"}]}`
	request := claudeCodeMessageRequest(body)
	if err := applyClientEmulationBody(model.ClientEmulationClaudeCode, request); err != nil {
		t.Fatal(err)
	}
	if string(request.Body) != body {
		t.Fatalf("valid body was rewritten:\n%s", request.Body)
	}
}

// TestShapeClaudeCodeSystemForms system 的四种输入形态都要被规范成上游认可的形状。
func TestShapeClaudeCodeSystemForms(t *testing.T) {
	identity := func(block gjson.Result) bool {
		return block.Get("text").String() == claudeCodeIdentityLine
	}

	cases := []struct {
		name   string
		system string
		check  func(t *testing.T, root gjson.Result)
	}{
		{
			name:   "string system becomes array with identity first",
			system: `"You are a helpful assistant."`,
			check: func(t *testing.T, root gjson.Result) {
				blocks := root.Get("system").Array()
				if len(blocks) != 2 || !identity(blocks[0]) || blocks[1].Get("text").String() != "You are a helpful assistant." {
					t.Fatalf("blocks: %s", root.Get("system").Raw)
				}
			},
		},
		{
			name:   "ordinary first block gets identity prepended",
			system: `[{"type":"text","text":"project context"}]`,
			check: func(t *testing.T, root gjson.Result) {
				blocks := root.Get("system").Array()
				if len(blocks) != 2 || !identity(blocks[0]) || blocks[1].Get("text").String() != "project context" {
					t.Fatalf("blocks: %s", root.Get("system").Raw)
				}
			},
		},
		{
			name:   "billing header keeps position, identity inserted after",
			system: `[{"type":"text","text":"x-anthropic-billing-header: cc_version=2.1.296.406; cc_entrypoint=sdk-cli;"},{"type":"text","text":"big prompt"}]`,
			check: func(t *testing.T, root gjson.Result) {
				blocks := root.Get("system").Array()
				if len(blocks) != 3 {
					t.Fatalf("blocks: %s", root.Get("system").Raw)
				}
				if !strings.HasPrefix(blocks[0].Get("text").String(), "x-anthropic-billing-header:") {
					t.Fatalf("billing block moved: %s", blocks[0].Raw)
				}
				if !identity(blocks[1]) || blocks[2].Get("text").String() != "big prompt" {
					t.Fatalf("blocks: %s", root.Get("system").Raw)
				}
			},
		},
		{
			name:   "identity first block untouched",
			system: `[{"type":"text","text":"` + claudeCodeIdentityLine + `","cache_control":{"type":"ephemeral"}},{"type":"text","text":"context"}]`,
			check: func(t *testing.T, root gjson.Result) {
				blocks := root.Get("system").Array()
				if len(blocks) != 2 || blocks[0].Get("cache_control.type").String() != "ephemeral" {
					t.Fatalf("blocks: %s", root.Get("system").Raw)
				}
			},
		},
		{
			name:   "agent sdk line is an accepted identity too",
			system: `[{"type":"text","text":"` + claudeCodeAgentSDKLine + `"}]`,
			check: func(t *testing.T, root gjson.Result) {
				if len(root.Get("system").Array()) != 1 {
					t.Fatalf("blocks: %s", root.Get("system").Raw)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(`{"model":"m","system":` + tc.system + `,"messages":[{"role":"user","content":"hi"}]}`)
			shaped, err := shapeClaudeCodeSystem(body)
			if err != nil {
				t.Fatal(err)
			}
			tc.check(t, gjson.ParseBytes(shaped))
			// 其余字段逐字节保留。
			if got := gjson.GetBytes(shaped, "model").String(); got != "m" {
				t.Fatalf("model lost: %q", got)
			}
			if got := gjson.GetBytes(shaped, "messages.0.role").String(); got != "user" {
				t.Fatalf("messages lost: %q", got)
			}
		})
	}
}

// TestShapeClaudeCodeMetadataForms metadata.user_id 的合法性判定与替换。
func TestShapeClaudeCodeMetadataForms(t *testing.T) {
	validV2 := `{"device_id":"` + strings.Repeat("a", 64) + `","account_uuid":"","session_id":"61ed3771-7779-4014-8249-b39daeebd88c"}`
	validLegacy := "user_" + strings.Repeat("b", 64) + "_account__session_61ed3771-7779-4014-8249-b39daeebd88c"

	keep := []struct {
		name   string
		userID string
	}{
		{"valid v2 kept", validV2},
		{"legacy kept", validLegacy},
		{"v2 without account_uuid kept", `{"device_id":"` + strings.Repeat("c", 64) + `","session_id":"61ed3771-7779-4014-8249-b39daeebd88c"}`},
	}
	for _, tc := range keep {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{"user_id": tc.userID})
			if err != nil {
				t.Fatal(err)
			}
			body := []byte(`{"model":"m","metadata":` + string(raw) + `}`)
			shaped, err := shapeClaudeCodeMetadata(body)
			if err != nil {
				t.Fatal(err)
			}
			if got := gjson.GetBytes(shaped, "metadata.user_id").String(); got != tc.userID {
				t.Fatalf("valid user_id rewritten: %q", got)
			}
		})
	}

	replace := []struct {
		name string
		body string
	}{
		{"missing metadata", `{"model":"m"}`},
		{"missing user_id", `{"model":"m","metadata":{"other":1}}`},
		{"empty user_id", `{"model":"m","metadata":{"user_id":""}}`},
		{"non-json string", `{"model":"m","metadata":{"user_id":"user_123"}}`},
		{"v2 missing session", `{"model":"m","metadata":{"user_id":"{\"device_id\":\"` + strings.Repeat("a", 64) + `\"}"}}`},
		{"session not uuid", `{"model":"m","metadata":{"user_id":"{\"device_id\":\"` + strings.Repeat("a", 64) + `\",\"session_id\":\"` + strings.Repeat("b", 36) + `\"}"}}`},
		{"legacy bad session", `{"model":"m","metadata":{"user_id":"user_` + strings.Repeat("d", 64) + `_account__session_not-a-uuid"}}`},
	}
	for _, tc := range replace {
		t.Run(tc.name, func(t *testing.T) {
			shaped, err := shapeClaudeCodeMetadata([]byte(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			userID := gjson.GetBytes(shaped, "metadata.user_id").String()
			if !claudeCodeUserIDValid(userID) {
				t.Fatalf("invalid user_id not replaced: %q", userID)
			}
			if got := gjson.GetBytes(shaped, "model").String(); got != "m" {
				t.Fatalf("model lost: %q", got)
			}
		})
	}
}

// TestApplyClientEmulationBodyScope 体层只在 Claude Code + Messages 时生效。
//
// 其余模式 (不伪装 / Codex) 与非 Messages 端点必须一字节都不改动 ——
// 否则上线即改变所有渠道的出站体形状, 而只有 CC + Messages 这套 system/metadata 语义成立。
func TestApplyClientEmulationBodyScope(t *testing.T) {
	raw := `{"model":"m","messages":[{"role":"user","content":"hi"}]}`

	cases := []struct {
		name      string
		emulation model.ClientEmulation
		format    string
		body      string
	}{
		{"none emulation", model.ClientEmulationNone, string(llm.APIFormatAnthropicMessage), raw},
		{"codex emulation", model.ClientEmulationCodex, string(llm.APIFormatAnthropicMessage), raw},
		{"chat completion endpoint", model.ClientEmulationClaudeCode, string(llm.APIFormatOpenAIChatCompletion), raw},
		{"responses endpoint", model.ClientEmulationClaudeCode, string(llm.APIFormatOpenAIResponse), raw},
		{"non json body", model.ClientEmulationClaudeCode, string(llm.APIFormatAnthropicMessage), `not json`},
		{"non object body", model.ClientEmulationClaudeCode, string(llm.APIFormatAnthropicMessage), `[1,2]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := &httpclient.Request{APIFormat: tc.format, Body: []byte(tc.body)}
			if err := applyClientEmulationBody(tc.emulation, request); err != nil {
				t.Fatal(err)
			}
			if string(request.Body) != tc.body {
				t.Fatalf("body changed: %s -> %s", tc.body, request.Body)
			}
		})
	}
}

// TestApplyClientEmulationBodySyncsJSONBody JSONBody 非空时必须与 Body 同步。
//
// JSONBody 是给日志与后续处理用的副本; 只改 Body 会让两个视图分歧,
// 日志里看到的请求体与实际发出的不同。
func TestApplyClientEmulationBodySyncsJSONBody(t *testing.T) {
	body := []byte(`{"model":"m","messages":[{"role":"user","content":"hi"}]}`)
	request := &httpclient.Request{
		APIFormat: string(llm.APIFormatAnthropicMessage),
		Body:      body,
		JSONBody:  body,
	}
	if err := applyClientEmulationBody(model.ClientEmulationClaudeCode, request); err != nil {
		t.Fatal(err)
	}
	if string(request.JSONBody) != string(request.Body) {
		t.Fatalf("JSONBody out of sync:\nbody=%s\njson=%s", request.Body, request.JSONBody)
	}
	if !gjson.GetBytes(request.JSONBody, "system").Exists() {
		t.Fatalf("JSONBody missing shaped system: %s", request.JSONBody)
	}
}

// TestApplyChannelConfigAddsEmulationBody 整链: applyChannelConfig 在参数覆盖前补齐体形状。
//
// 顺序契约与头层一致: 伪装给默认值, 渠道的 param_override 最后定稿 ——
// override 显式设置的 system/metadata 必须赢过伪装补齐的。
func TestApplyChannelConfigAddsEmulationBody(t *testing.T) {
	request := claudeCodeMessageRequest(`{"model":"m","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	channel := model.Channel{ChannelConfig: model.ChannelConfig{ClientEmulation: model.ClientEmulationClaudeCode}}
	if err := applyChannelConfig(channel, request); err != nil {
		t.Fatal(err)
	}
	if !gjson.GetBytes(request.Body, "system").IsArray() {
		t.Fatalf("system not shaped: %s", request.Body)
	}
	if !claudeCodeUserIDValid(gjson.GetBytes(request.Body, "metadata.user_id").String()) {
		t.Fatalf("metadata not shaped: %s", request.Body)
	}

	// 渠道显式覆盖 system 时保持优势: 覆盖发生在伪装之后。
	request = claudeCodeMessageRequest(`{"model":"m","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	channel = model.Channel{ChannelConfig: model.ChannelConfig{
		ClientEmulation: model.ClientEmulationClaudeCode,
		ParamOverride:   `{"system":[{"type":"text","text":"channel defined"}]}`,
	}}
	if err := applyChannelConfig(channel, request); err != nil {
		t.Fatal(err)
	}
	if got := gjson.GetBytes(request.Body, "system.0.text").String(); got != "channel defined" {
		t.Fatalf("param override did not win: %s", request.Body)
	}
}

// TestEmulatedClientUsesStreaming 只有 CC 伪装的测活需要强制流式。
func TestEmulatedClientUsesStreaming(t *testing.T) {
	if !emulatedClientUsesStreaming(model.ClientEmulationClaudeCode) {
		t.Fatal("claude code emulation should force streaming probes")
	}
	if emulatedClientUsesStreaming(model.ClientEmulationNone) || emulatedClientUsesStreaming(model.ClientEmulationCodex) {
		t.Fatal("only claude code emulation forces streaming")
	}
}

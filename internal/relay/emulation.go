package relay

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/model"
)

// 本文件承载"客户端伪装": 部分上游 (尤为 Claude Code / Codex 中转) 会校验客户端身份,
// 按 User-Agent / originator / beta 头等判定请求是否来自官方客户端, 缺这些头时直接拒绝
// (如 new-api 系中转对 codex 通道要求 originator 与 OpenAI-Beta, 对 Claude 通道要求
// 1M 上下文 beta 头)。伪装只往出站请求上补身份头, 不改 URL, 协议与请求体 ——
// 走哪条端点仍由授权协议位的勾选决定。
//
// 头集合的取值来源:
//   - Claude Code: claude-cli 的 User-Agent 与 x-app 头; 1M 上下文 beta 头
//     (context-1m-2025-08-07) 按实测必要性默认带上 —— 缺它时 anyrouter 类中转以
//     "请启用 1m 上下文后重试" 拒绝, 而该 beta 对不支持的上游只是被忽略的额外头。
//   - Codex: 参照 axonhub 中已验证可用的 Codex CLI 会话形状
//     (originator / OpenAI-Beta / Session / Thread / Turn 元数据)。
const (
	claudeCodeUserAgent = "claude-cli/2.0.0 (external, cli)"
	claudeCodeVersion   = "2023-06-01"
	// claudeCodeContext1MBeta 是 1M 上下文的 beta 头值; 默认带上, 见上方说明。
	claudeCodeContext1MBeta = "context-1m-2025-08-07"

	codexOriginator = "codex_cli_rs"
	codexVersion    = "0.159.0"
	codexBeta       = "responses=experimental"
)

// ApplyClientEmulation 按渠道的伪装模式补齐出站身份头, 直接改写传入的 Header。
// 转发与测活经由 applyChannelConfig 统一走这里; 模型拉取 (拉 /models) 不在转发管线上,
// 由 handlers 显式调用本函数补同样的头, 保证"编辑表单里试拉"与"保存后转发"看到的身份一致。
//
// 调用方必须把本函数放在自定义 Header 之前调用: 伪装给的是"官方客户端的默认形状",
// 渠道的 custom_header 应能覆盖其中任何一项 (包括 beta 头), 故顺序是伪装先铺底、自定义最后覆盖。
//
// 对已存在的头: 客户端或转换器已经指定过的身份头不覆盖; anthropic-beta 是叠加语义,
// 已有其他 beta 值时追加 1M 而不丢原有的。
func ApplyClientEmulation(emulation model.ClientEmulation, headers http.Header) error {
	if headers == nil {
		return nil
	}
	switch emulation {
	case model.ClientEmulationNone:
		return nil
	case model.ClientEmulationClaudeCode:
		applyClaudeCodeEmulation(headers)
		return nil
	case model.ClientEmulationCodex:
		return applyCodexEmulation(headers)
	default:
		return fmt.Errorf("unsupported client emulation: %q", emulation)
	}
}

// applyClaudeCodeEmulation 按 Claude Code CLI 的形状补身份头。
func applyClaudeCodeEmulation(headers http.Header) {
	setIfEmpty(headers, "User-Agent", claudeCodeUserAgent)
	setIfEmpty(headers, "X-App", "cli")
	setIfEmpty(headers, "Anthropic-Version", claudeCodeVersion)
	// anthropic-beta 可叠加: 已有其他 beta 值时追加 1M, 不丢原有的。
	appendHeaderToken(headers, "Anthropic-Beta", claudeCodeContext1MBeta)
}

// applyCodexEmulation 按 Codex CLI 的形状补身份头与会话头。
// 会话头每次调用现生成: 伪装层没有跨请求的会话概念, 而上游只要求"形状完整"。
func applyCodexEmulation(headers http.Header) error {
	setIfEmpty(headers, "Originator", codexOriginator)
	setIfEmpty(headers, "User-Agent", codexOriginator+"/"+codexVersion)
	setIfEmpty(headers, "Version", codexVersion)
	setIfEmpty(headers, "OpenAI-Beta", codexBeta)

	sessionID := newEmulationSessionID()
	setIfEmpty(headers, "Session-Id", sessionID)
	setIfEmpty(headers, "Thread-Id", sessionID)
	setIfEmpty(headers, "Conversation_id", sessionID)
	setIfEmpty(headers, "X-Codex-Window-Id", sessionID+":0")
	setIfEmpty(headers, "X-Client-Request-Id", sessionID)
	// Turn 元数据: 上游按它归组一次"对话轮次", 形状与 Codex 客户端一致即可。
	turnMetadata, err := json.Marshal(map[string]any{
		"session_id":              sessionID,
		"thread_id":               sessionID,
		"turn_id":                 sessionID,
		"window_id":               sessionID + ":0",
		"request_kind":            "turn",
		"thread_source":           "user",
		"sandbox":                 "none",
		"turn_started_at_unix_ms": time.Now().UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("marshal codex turn metadata: %w", err)
	}
	setIfEmpty(headers, "X-Codex-Turn-Metadata", string(turnMetadata))
	return nil
}

// newEmulationSessionID 造一个固定长度的会话标识: 由纳秒时间戳散列而来, 每次调用不同。
// 不需要真随机: 上游只拿它做会话归组, 不参与鉴权; 用哈希只为得到一个固定长度、字符集安全的串。
func newEmulationSessionID() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(sum[0:4]),
		binary.BigEndian.Uint16(sum[4:6]),
		binary.BigEndian.Uint16(sum[6:8]),
		binary.BigEndian.Uint16(sum[8:10]),
		sum[10:16],
	)
}

// setIfEmpty 仅在头不存在时写入, 保留已有值。
func setIfEmpty(headers http.Header, key, value string) {
	if headers.Get(key) == "" {
		headers.Set(key, value)
	}
}

// appendHeaderToken 把值按逗号分隔追加进列表型头 (如 anthropic-beta), 已存在则不重复加。
func appendHeaderToken(headers http.Header, key, token string) {
	current := headers.Get(key)
	if current == "" {
		headers.Set(key, token)
		return
	}
	for _, existing := range strings.Split(current, ",") {
		if strings.TrimSpace(existing) == token {
			return
		}
	}
	headers.Set(key, current+","+token)
}

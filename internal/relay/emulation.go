package relay

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/bestruirui/octopus/internal/model"
)

// 本文件承载"客户端伪装": 部分上游 (尤为 Claude Code / Codex 中转) 会校验客户端身份,
// 按请求头与请求体的形状判定请求是否来自官方客户端, 缺项时直接拒绝
// (如 new-api 系中转对 codex 通道要求 originator 与 OpenAI-Beta, 对 Claude 通道要求
// 1M 上下文 beta 头与 CC 形状的请求体)。
//
// 伪装分两层, 都不改 URL 与协议方向 —— 走哪条端点仍由授权协议位的勾选决定:
//   - 头层 (ApplyClientEmulation): 按官方客户端形状补身份头;
//   - 体层 (applyClientEmulationBody): 仅 Claude Code + Messages 端点, 补齐请求体里的
//     身份形状 (system 首块标记行 + metadata.user_id)。
//
// 头集合的取值来源:
//   - Claude Code: claude-cli 的 User-Agent 与 x-app 头; 1M 上下文 beta 头
//     (context-1m-2025-08-07) 按实测必要性默认带上 —— 缺它时任何 Messages 请求被
//     anyrouter 类中转以"请启用 1m 上下文后重试"直接 400 拒绝, 而该 beta 对不支持的
//     上游只是被忽略的额外头。
//   - Codex: 参照 axonhub 中已验证可用的 Codex CLI 会话形状
//     (originator / OpenAI-Beta / Session / Thread / Turn 元数据)。
//
// 体层 (Claude Code) 的取值来源: 用真机 Claude Code (2.1.296) 的请求经 mock 上游捕获后
// 逐项消融。对 anyrouter 的 fable 类模型 (供给受限、被中转优先保障 CC 客户端):
//   - system 必须是数组, 且首块文本是官方标记行之一 (计费头 "x-anthropic-billing-header:"
//     打头、CC 身份行、Agent SDK 行); 字符串形态或首块为普通文本一律被 503 拒绝;
//   - metadata.user_id 必须是可解析的 v2 JSON (device_id 与合法 UUID 形式的 session_id
//     都非空; account_uuid 可空) 或 legacy 行 (user_<64hex>_account__session_<uuid>);
//     缺失或不可解析即 503。
//     两项都是硬门槛: 任一缺失时真机完整体请求成功率 100% -> 轻体(缺形状) 0% (同分钟交错实测)。
const (
	claudeCodeUserAgent = "claude-cli/2.0.0 (external, cli)"
	claudeCodeVersion   = "2023-06-01"
	// claudeCodeContext1MBeta 是 1M 上下文的 beta 头值; 默认带上, 见上方说明。
	claudeCodeContext1MBeta = "context-1m-2025-08-07"

	// claudeCodeIdentityLine 是 Claude Code 官方身份行的经典形态 (CC 与 axonhub 同款);
	// 作为 system 首块被上游认可, 且对模型只增加一行"你是 CLI"的角色声明, 不改变任务语义。
	claudeCodeIdentityLine = "You are Claude Code, Anthropic's official CLI for Claude."
	// claudeCodeAgentSDKLine 是新版 CC (2.1.x) system 里另一条被认可的标记行。
	claudeCodeAgentSDKLine = "You are a Claude agent, built on Anthropic's Claude Agent SDK."
	// claudeCodeBillingHeaderPrefix 是 CC 计费行前缀; 真机上 system 首块常以它打头。
	claudeCodeBillingHeaderPrefix = "x-anthropic-billing-header:"

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

// emulatedClientUsesStreaming 报告该伪装模式的测活是否必须以流式发出。
//
// Claude Code 客户端对 Messages 端点永远使用流式 (官方实现如此); 更关键的是实测:
// anyrouter 类中转的 CC 供给路径对非流式请求不回内容 (HTTP 520), 流式才能触达真实上游。
// 测活要"对应 Claude Code 客户端", 就必须按 CC 的流式形状测 —— 否则这些模型永远显示不通,
// 而真实客户端 (永远流式) 明明能通。Codex 与不伪装不受影响。
func emulatedClientUsesStreaming(emulation model.ClientEmulation) bool {
	return emulation == model.ClientEmulationClaudeCode
}

// applyClaudeCodeEmulation 按 Claude Code CLI 的形状补身份头。
func applyClaudeCodeEmulation(headers http.Header) {
	setIfEmpty(headers, "User-Agent", claudeCodeUserAgent)
	setIfEmpty(headers, "X-App", "cli")
	setIfEmpty(headers, "Anthropic-Version", claudeCodeVersion)
	// anthropic-beta 可叠加: 已有其他 beta 值时追加 1M, 不丢原有的。
	appendHeaderToken(headers, "Anthropic-Beta", claudeCodeContext1MBeta)
}

// applyClientEmulationBody 按伪装模式补齐请求体里的身份形状。
//
// 仅 Claude Code 模式且目标端点为 Messages 时生效: 其余模式的渠道出站体必须与旧版逐字节一致,
// 非 Messages 的端点 (Chat/Responses) 也没有这套 system/metadata 语义。体不是 JSON 对象
// (极少见的非常规负载) 时原样放行 —— 宁可让上游按原来的方式拒绝, 也不在伪装层里报错。
//
// 补齐只在缺失或不合形状时发生, 已有合法值一律保留 (见 shapeClaudeCodeSystem/shapeClaudeCodeMetadata):
// 真实 Claude Code 客户端经本网关转发时, 请求体原样通过, 不被改写。
func applyClientEmulationBody(emulation model.ClientEmulation, request *httpclient.Request) error {
	if emulation != model.ClientEmulationClaudeCode || request == nil || len(request.Body) == 0 {
		return nil
	}
	if request.APIFormat != string(llm.APIFormatAnthropicMessage) || !json.Valid(request.Body) {
		return nil
	}
	if !gjson.ParseBytes(request.Body).IsObject() {
		return nil
	}

	body, err := shapeClaudeCodeSystem(request.Body)
	if err != nil {
		return err
	}
	body, err = shapeClaudeCodeMetadata(body)
	if err != nil {
		return err
	}
	if string(body) == string(request.Body) {
		return nil
	}
	request.Body = body
	// JSONBody 是给日志与后续处理用的副本, 非空时必须同步, 否则两个视图分歧。
	if len(request.JSONBody) > 0 {
		request.JSONBody = slices.Clone(body)
	}
	return nil
}

// shapeClaudeCodeSystem 把 system 规范成上游认可的形状。
//
// 实测 (anyrouter fable 类模型, 逐块消融):
//   - 无 system / system 为普通字符串 / 首块为普通文本: 一律 503;
//   - 首块为 CC 身份行 ("You are Claude Code...") 或 Agent SDK 行: 通过;
//   - 首块为计费头 ("x-anthropic-billing-header:" 打头) 时, 第二块必须是身份行 (真机形状);
//     计费头后直接接普通文本同样 503, 而身份行在计费头之前也 503 —— 计费块的位置是固定的。
//
// 于是: 首块已是身份行则原样保留; 首块是计费头则把身份行插到它后面; 其余情况把身份行前置。
// 只动 system 一项, 其余字段逐字节保留 (sjson 只改写目标路径)。
func shapeClaudeCodeSystem(body []byte) ([]byte, error) {
	system := gjson.GetBytes(body, "system")
	if !system.Exists() {
		return setSystemBlocks(body, []json.RawMessage{mustIdentityBlock()})
	}
	if system.Type == gjson.String {
		// 字符串形态即使内容里含身份行也不被认可 (实测 503); 转成 [身份行, 原文] 的数组形态。
		original, err := json.Marshal(system.String())
		if err != nil {
			return body, nil
		}
		block := []byte(`{"type":"text","text":` + string(original) + `}`)
		return setSystemBlocks(body, []json.RawMessage{mustIdentityBlock(), block})
	}
	if !system.IsArray() {
		// 非常规形态 (如对象): 整体替换成身份行单块, 保证可被上游接受。
		return setSystemBlocks(body, []json.RawMessage{mustIdentityBlock()})
	}

	blocks := system.Array()
	if len(blocks) == 0 {
		return setSystemBlocks(body, []json.RawMessage{mustIdentityBlock()})
	}
	firstText := blocks[0].Get("text").String()
	switch {
	case isClaudeCodeIdentityText(firstText):
		return body, nil
	case isClaudeCodeBillingText(firstText):
		if len(blocks) > 1 && isClaudeCodeIdentityText(blocks[1].Get("text").String()) {
			return body, nil
		}
		return insertSystemBlock(body, blocks, 1)
	default:
		return insertSystemBlock(body, blocks, 0)
	}
}

// isClaudeCodeIdentityText 判断一段 system 文本是否为被上游认可的 CC 身份标记行。
func isClaudeCodeIdentityText(text string) bool {
	return text == claudeCodeIdentityLine || text == claudeCodeAgentSDKLine
}

// isClaudeCodeBillingText 判断一段 system 文本是否为 CC 计费头行。
func isClaudeCodeBillingText(text string) bool {
	return strings.HasPrefix(text, claudeCodeBillingHeaderPrefix)
}

// mustIdentityBlock 生成身份行的 JSON 编码; 固定内容, marshal 不会失败, 失败也只可能源于进程级异常。
func mustIdentityBlock() json.RawMessage {
	block, err := json.Marshal(map[string]string{"type": "text", "text": claudeCodeIdentityLine})
	if err != nil {
		panic("marshal claude code identity block: " + err.Error())
	}
	return block
}

// setSystemBlocks 用给定块序列整体替换 system。
func setSystemBlocks(body []byte, blocks []json.RawMessage) ([]byte, error) {
	encoded, err := json.Marshal(blocks)
	if err != nil {
		return nil, fmt.Errorf("marshal claude code system blocks: %w", err)
	}
	return sjson.SetRawBytes(body, "system", encoded)
}

// insertSystemBlock 在 blocks 的 index 处插入身份行, 保留其余块逐字节原样。
func insertSystemBlock(body []byte, blocks []gjson.Result, index int) ([]byte, error) {
	merged := make([]json.RawMessage, 0, len(blocks)+1)
	for position, block := range blocks {
		if position == index {
			merged = append(merged, mustIdentityBlock())
		}
		merged = append(merged, json.RawMessage(block.Raw))
	}
	if index >= len(blocks) {
		merged = append(merged, mustIdentityBlock())
	}
	return setSystemBlocks(body, merged)
}

// claudeCodeLegacyUserIDPattern 匹配旧版 CC user_id 行: user_<64hex>_account__session_<uuid>。
var claudeCodeLegacyUserIDPattern = regexp.MustCompile(`^user_([a-fA-F0-9]{64})_account__session_(.+)$`)

// claudeCodeUUIDPattern 匹配标准 UUID 文本 (8-4-4-4-12 十六进制), 不校验版本与变体位 ——
// 上游对 session_id 的判定同样只看形状 (实测: 'b'*36 被拒, 任意合法 UUID 文本都通过)。
var claudeCodeUUIDPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// shapeClaudeCodeMetadata 保证 metadata.user_id 是上游可解析的形态; 不可解析时换成新生成的 v2 JSON。
//
// 实测: user_id 缺失、或 device_id/session_id 缺失、或 session_id 不是合法 UUID 一律 503;
// 合法 v2 (account_uuid 可空) 与合法 legacy 行通过。已合法的一律保留 —— 真实 CC 转发时原样传递。
func shapeClaudeCodeMetadata(body []byte) ([]byte, error) {
	metadata := gjson.GetBytes(body, "metadata")
	if metadata.Exists() && !metadata.IsObject() {
		// metadata 是非常规类型 (如字符串): 整体重置为对象, 再放 user_id。
		next, err := sjson.SetRawBytes(body, "metadata", []byte(`{}`))
		if err != nil {
			return nil, fmt.Errorf("reset claude code metadata: %w", err)
		}
		body = next
	}
	if claudeCodeUserIDValid(gjson.GetBytes(body, "metadata.user_id").String()) {
		return body, nil
	}
	generated, err := newClaudeCodeUserID()
	if err != nil {
		return nil, err
	}
	next, err := sjson.SetBytes(body, "metadata.user_id", generated)
	if err != nil {
		return nil, fmt.Errorf("set claude code user id: %w", err)
	}
	return next, nil
}

// claudeCodeUserIDValid 判定 user_id 是否已是上游可解析的形态。
func claudeCodeUserIDValid(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if strings.HasPrefix(raw, "{") {
		var uid struct {
			DeviceID  string `json:"device_id"`
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal([]byte(raw), &uid); err != nil {
			return false
		}
		return uid.DeviceID != "" && claudeCodeUUIDPattern.MatchString(uid.SessionID)
	}
	if matches := claudeCodeLegacyUserIDPattern.FindStringSubmatch(raw); matches != nil {
		return claudeCodeUUIDPattern.MatchString(matches[2])
	}
	return false
}

// newClaudeCodeUserID 生成一份 v2 形态的 user_id: device_id 为 64 位十六进制随机值,
// account_uuid 留空 (实测可选), session_id 为随机 UUID 文本。每次调用都不同 ——
// 伪装层没有跨请求的会话概念, 上游也只按它做归组。
func newClaudeCodeUserID() (string, error) {
	device := make([]byte, 32)
	if _, err := rand.Read(device); err != nil {
		return "", fmt.Errorf("generate claude code device id: %w", err)
	}
	encoded, err := json.Marshal(struct {
		DeviceID    string `json:"device_id"`
		AccountUUID string `json:"account_uuid"`
		SessionID   string `json:"session_id"`
	}{
		DeviceID:    hex.EncodeToString(device),
		AccountUUID: "",
		SessionID:   newEmulationUUID(),
	})
	if err != nil {
		return "", fmt.Errorf("marshal claude code user id: %w", err)
	}
	return string(encoded), nil
}

// newEmulationUUID 从密码学随机源生成一个 UUID 文本 (版本与变体位按 RFC 4122 置位)。
// 不复用 newEmulationSessionID 的时间散列: user_id 是设备/会话标识, 用随机源更贴合语义。
func newEmulationUUID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		// 随机源失败是进程级异常; 退回时间散列仍能给出形状合法的 UUID,
		// 上游只校验形状, 不校验熵 —— 这里不值得为几字节随机性让一次转发失败。
		sum := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		copy(raw, sum[:16])
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(raw[0:4]),
		binary.BigEndian.Uint16(raw[4:6]),
		binary.BigEndian.Uint16(raw[6:8]),
		binary.BigEndian.Uint16(raw[8:10]),
		raw[10:16],
	)
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

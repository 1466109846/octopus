<div align="center">

<img src="web/public/logo.svg" alt="Octopus Logo" width="120" height="120">

### Octopus

**为个人打造的简单、美观、优雅的 LLM API 聚合服务**

简体中文 | [English](README.md)

</div>


## ✨ 特性

- 🔀 **多渠道聚合** - 支持接入多个 LLM 供应商渠道，统一管理
- 🔄 **协议互转** - 支持 OpenAI Chat / OpenAI Responses / Anthropic 三种 API 格式互相转换
- 💰 **价格同步** - 自动更新模型价格
- 🔃 **模型同步** - 自动与渠道同步可用模型列表，省心省力
- 🛡️ **自动故障转移** - 上游渠道发生故障时自动切换到可用渠道
- 🔍 **请求全链路实时可视化** - 客户端发起请求后，即可在前端实时查看完整请求链路
- 🚧 **上游错误拦截** - 拦截所有上游错误，避免中断 Agent 任务
- 📊 **数据统计** - 全面的请求统计、Token 消耗、费用追踪
- 🎨 **优雅界面** - 简洁美观的 Web 管理面板
- 📦 **轻量单文件部署** - 单个二进制文件即可运行，无需额外运行时依赖
- 🗄️ **多数据库支持** - 支持 SQLite、MySQL、PostgreSQL
- 🎭 **客户端伪装** - 按渠道伪装 Claude Code / Codex 客户端身份（头 + 请求体形状），应对校验客户端身份的上游；测活与糖果测试同形状走流式


## 🚀 快速开始

### 🐳 Docker 运行

本次 Release 只发布二进制，不发布 GHCR 镜像。要使用 Docker，请在本地构建本 fork 的镜像：

```bash
git clone https://github.com/1466109846/octopus.git
cd octopus
docker build -f scripts/dockerfile/Dockerfile -t octopus:local .
docker run -d --name octopus -v /path/to/data:/app/data -p 18874:18874 octopus:local
```

或者在本仓库目录下用 docker compose 本地构建运行：

```bash
docker compose up -d --build
```


### 📦 从 Release 下载

从 [Releases](https://github.com/1466109846/octopus/releases) 下载对应平台的二进制文件，然后运行：

```bash
./octopus start
```

### 🛠️ 源码运行

**环境要求：**
- Go 1.26.4
- Node.js 24+
- pnpm 10+

```bash
# 克隆本分支项目
git clone https://github.com/1466109846/octopus.git
cd octopus
# 构建前端
cd web && pnpm install && pnpm run build
# 启动后端服务
go run main.go start 
```

> 💡 **提示**：前端构建产物会被嵌入到 Go 二进制文件中，所以必须先构建前端再启动后端。

**开发模式**

```bash
cd web && pnpm install && pnpm run dev
## 新建终端,启动后端服务
go run main.go start
## 访问前端地址
http://localhost:5173
```

### 🔐 默认账户

首次启动后，访问 http://localhost:18874 使用以下默认账户登录管理面板：

- **用户名**：`admin`
- **密码**：`admin`

> ⚠️ **安全提示**：请在首次登录后立即修改默认密码。

### 📝 配置文件

配置文件默认位于 `data/config.json`，首次启动时自动生成。

**完整配置示例：**

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 18874
  },
  "database": {
    "type": "sqlite",
    "path": "data/data.db"
  },
  "log": {
    "level": "info"
  }
}
```

**配置项说明：**

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.host` | 监听地址 | `0.0.0.0` |
| `server.port` | 服务端口 | `18874` |
| `database.type` | 数据库类型 | `sqlite` |
| `database.path` | 数据库连接地址 | `data/data.db` |
| `log.level` | 日志级别 | `info` |

**数据库配置：**

支持三种数据库：

| 类型 | `database.type` | `database.path` 格式 |
|------|-----------------|---------------------|
| SQLite | `sqlite` | `data/data.db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/dbname` |
| PostgreSQL | `postgres` | `postgresql://user:password@host:port/dbname?sslmode=disable` |

**MySQL 配置示例：**

```json
{
  "database": {
    "type": "mysql",
    "path": "root:password@tcp(127.0.0.1:3306)/octopus"
  }
}
```

**PostgreSQL 配置示例：**

```json
{
  "database": {
    "type": "postgres",
    "path": "postgresql://user:password@localhost:5432/octopus?sslmode=disable"
  }
}
```

> 💡 **提示**：MySQL 和 PostgreSQL 需要先手动创建数据库，程序会自动创建表结构。

**环境变量：**

所有配置项均可通过环境变量覆盖，格式为 `OCTOPUS_` + 配置路径（用 `_` 连接）：

| 环境变量 | 对应配置项 |
|----------|-----------|
| `OCTOPUS_SERVER_PORT` | `server.port` |
| `OCTOPUS_SERVER_HOST` | `server.host` |
| `OCTOPUS_DATABASE_TYPE` | `database.type` |
| `OCTOPUS_DATABASE_PATH` | `database.path` |
| `OCTOPUS_LOG_LEVEL` | `log.level` |
| `OCTOPUS_GITHUB_PAT` | 用于获取最新版本时的速率限制(可选) |


## 📸 界面预览

### 🖥️ 桌面端

<div align="center">
<table>
<tr>
<td align="center"><b>首页</b></td>
<td align="center"><b>渠道</b></td>
<td align="center"><b>分组</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-home.png" alt="首页" width="400"></td>
<td><img src="web/public/screenshot/desktop-channel.png" alt="渠道" width="400"></td>
<td><img src="web/public/screenshot/desktop-group.png" alt="分组" width="400"></td>
</tr>
<tr>
<td align="center"><b>价格</b></td>
<td align="center"><b>日志</b></td>
<td align="center"><b>设置</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-price.png" alt="价格" width="400"></td>
<td><img src="web/public/screenshot/desktop-log.png" alt="日志" width="400"></td>
<td><img src="web/public/screenshot/desktop-setting.png" alt="设置" width="400"></td>
</tr>
</table>
</div>

### 📱 移动端

<div align="center">
<table>
<tr>
<td align="center"><b>首页</b></td>
<td align="center"><b>渠道</b></td>
<td align="center"><b>分组</b></td>
<td align="center"><b>价格</b></td>
<td align="center"><b>日志</b></td>
<td align="center"><b>设置</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/mobile-home.png" alt="移动端首页" width="140"></td>
<td><img src="web/public/screenshot/mobile-channel.png" alt="移动端渠道" width="140"></td>
<td><img src="web/public/screenshot/mobile-group.png" alt="移动端分组" width="140"></td>
<td><img src="web/public/screenshot/mobile-price.png" alt="移动端价格" width="140"></td>
<td><img src="web/public/screenshot/mobile-log.png" alt="移动端日志" width="140"></td>
<td><img src="web/public/screenshot/mobile-setting.png" alt="移动端设置" width="140"></td>
</tr>
</table>
</div>


## 📖 功能说明

> 📢 **本仓库是 `bestruirui/octopus` 的一个分支(fork)**，在上游基础上新增并维护了以下功能。

### 🕵️ 模型监控 / 定时测活

定时探测保活（scheduled probe）：在后台按节奏自动探测分组里每一个渠道·凭据的可用性与延迟，结论实时回显在**模型监控**页：

- 支持多目标并更名：一个任务可监控多条渠道/模型，分组名称可自定义
- 间隔按凭据均摊：探测间隔会自动在凭据之间均匀分配，避免同组探测撞车
- 行序可拖拽：拖动行改变探测顺序
- 结论不再过期：上次探测通过/失败的徽标与耗时实时滚动，过期结论在 5 分钟后自动置灰
- 仅 fork 生效：该模块在上游 master 不存在

### 🧪 上游掺水检测（响应模型对比）

> **动机**：上游（尤其各类中转站）可能偷换模型回话——你请求 `claude-opus-4-8`，实际回话的却是别的模型，而客户端只看到自己请求的名字。账号花了旗舰模型的钱、拿到却可能是廉价模型的服务，这类"掺水"在社区已有大量案例，但此前没有任何本地手段能直接看见。

日志页现在会自动对比**请求的模型**与**上游响应正文里声称服务的模型**（解析响应 JSON 顶层的 `model` 字段）：

- **一致**：绿色对勾确认；**不一致**：红色警告徽标 + 上游实际回报的模型名，悬停可见完整说明（卡片与详情弹窗两处都显示）
- 覆盖全部四条转发路径（同协议透传 / 跨协议转换 × 流式 / 非流式）——最终正文都汇流到请求状态的定稿点，解析只做一次
- 上游未给该字段时不展示，不误报；比较忽略大小写与首尾空白，不因边界噪声报警
- **只对比、不拦截**：部分中转的"模型名映射"属于正常行为（官网名与 API 名不同），是否处置由使用者按日志判断

### 🛡️ 测活提示词防封号

> **动机**：通用的 `"hi"` 测活文案容易被上游站点识别为探针流量，有封号风险（已有站点对测活流量明确提示/限制）。账号来之不易，测活是本程序的核心功能，不能因此没法用。

测活请求改发一段人类语气的通信测试话术：

> 喂喂喂 通信测试通信测试：请回答1988 收到请回复：是的，1988

模型看到后会尽快回一句短应答，维持原有的最小成本路径（`max_tokens` 仍为 16，超时与结论语义完全不变）。若哪天该话术也被识别，只需改 `internal/relay/probe.go` 里的 `probePrompt` 一行、重新构建替换二进制即可。

### 💓 保活

> **动机**：有些上游需要你一直"在场"——排队制或活跃度限流的服务，一段时间没有流量就把你踢出队列。模型监控回答的是"这条通道健不健康"，与保活不是同一件事：监控失败会冷却渠道，而排队场景里被拒（HTTP 429）是常态，照监控语义处理会把一条正常的渠道越保越死。

**保活**页按间隔向选定的渠道·凭据发送最小请求（默认 **5 分钟**），尝试维持上游活跃与排队位：

- 队列、间隔均摊、凭据轮转与逐行展示全部与模型监控共用（同一张任务表，按任务种类分流），两个页面互不串台
- 每行展示最近一次心跳的状态、**HTTP 状态码**与耗时：`429/503` 表示请求已到达上游但被拒绝；无状态码通常表示超时、网络故障或本地配置错误。**收到状态码不能证明排队位已保留**，是否有效取决于上游规则
- **对着"忙"持续敲门**：收到忙类拒绝（`429` 队列满 / `500`、`502`、`503`、`504` 上游瞬时故障）时，同一轮内自动退避重试（最多 3 次、2s→4s、整轮预算 60s）——排队位一旦出现空位就能立刻挤进去，而不是干等下一个周期（默认 5 分钟）只敲一次门。结论会标注"重试 N 次仍被拒"，一眼看出这套健壮性有没有生效
- 重试的边界：超时与网络错误（无状态码）不重试——那只会把一次挂起拖成三次；除 429 外的 4xx（钥匙不对/模型不存在等确定性拒绝）不重试，它们不会因为隔了两秒就变通
- **只展示、永不影响选路**：保活失败不冷却渠道、不降健康分、不参与故障转移——与排队场景的诉求正好相反，防止误伤正常渠道
- 保活固定发最小请求（不会发糖果题），且全天可测（不设时间窗），以减少夜间长时间无流量；无法保证上游不会清掉排队位
- 支持逐条或整任务的"立即保活一次"，逐条给出结论

### 🎭 客户端伪装（Claude Code / Codex）

> **动机**：一部分上游（尤为 Claude Code / Codex 系中转）会校验请求头里的客户端身份，判定请求是否来自官方客户端，不是就拒绝。实测两类典型拒绝：中转对缺少 1M 上下文 beta 头的请求直接回"请启用 1m 上下文后重试"；new-api 系的 codex 通道对缺少 `originator` / `OpenAI-Beta` 的请求回"invalid codex request"。网关自己发的请求天生不带这些头，于是测活与转发都会被拒——这与请求本身、协议、钥匙都无关，纯粹是"身份"问题。

渠道表单「连接」页新增**客户端伪装**下拉，按渠道选择出站请求伪装成哪种官方客户端：

- **不伪装（默认）**——保持网关自身的请求形状，行为与之前完全一致
- **Claude Code**——带上 `claude-cli` 的 User-Agent / `X-App: cli`，并**默认带上 1M 上下文 beta 头**（`context-1m-2025-08-07`）；`anthropic-beta` 按叠加处理，渠道已有其他 beta 值时追加而不是覆盖
- **Codex**——补齐 `originator: codex_cli_rs`、`OpenAI-Beta: responses=experimental`、`Version`、Session / Thread / Turn 元数据等一组 Codex CLI 会话形状的请求头

要点：

- **伪装分两层，都不改 URL 与协议方向**：头层补身份头，体层（仅 Claude Code + Messages）补请求体身份形状；走哪条端点（Chat / Responses / Messages）仍由授权协议位的勾选决定，与伪装正交
- 覆盖全部出站路径：转发（透传与转换）、测活（监控与保活）、模型拉取、图片转发——"表单里试拉"与"保存后转发"看到的客户端身份一致
- **顺序契约**：伪装先铺底，渠道的**自定义 Header 与参数覆盖最后生效**——想改用别的 UA / beta 组合 / system 时就显式覆盖
- 已存在的合法形状（如客户端透传的 User-Agent、真实 CC 客户端自己的 system 与 metadata）逐字节保留，伪装只补缺失项

**Claude Code 模式的两层伪装细节**（均为对照实测得出的硬门槛）：

- **头层**：`claude-cli` User-Agent / `X-App: cli`，并**默认带上 1M 上下文 beta 头**（`context-1m-2025-08-07`，缺它直接 400）；`anthropic-beta` 按叠加处理，渠道已有其他 beta 值时追加而不是覆盖
- **体层**（Messages 端点）：
  - `system` 规范为数组且首块为官方标记行（`You are Claude Code, Anthropic's official CLI for Claude.` 或 Agent SDK 行）；真机 2.1.x 的计费头（`x-anthropic-billing-header:`）打头时，身份行插到它后面
  - `metadata.user_id` 规范为可解析的 v2 JSON（`device_id` + 合法 UUID 的 `session_id`；`account_uuid` 可空）或 legacy 行——随机生成即可
  - 只对 Claude Code + Messages 生效；不开伪装的渠道与不支持 Messages 的端点一个字节都不动
- **测活按客户端形状走流式**：真实 Claude Code 对 Messages 永远流式，且上游的 CC 供给路径对非流式请求不回内容（实测 520 空响应）。因此 Claude Code 模式的可用性测活（监控 / 保活 / 模型页）与糖果测试均按流式发出；糖果测试沿事件流收集后用协议转换器聚合出完整正文再判分

### 📡 渠道管理

渠道是连接 LLM 供应商的基础配置单元。

**Base URL 说明：**

程序会根据渠道类型自动补全 API 版本和端点路径，您只需填写服务根地址即可：

| 渠道类型 | 自动补全路径 | 填写 URL | 完整请求地址示例 |
|----------|-------------|----------|-----------------|
| OpenAI Chat | `/v1/chat/completions` | `https://api.openai.com` | `https://api.openai.com/v1/chat/completions` |
| OpenAI Responses | `/v1/responses` | `https://api.openai.com` | `https://api.openai.com/v1/responses` |
| Anthropic | `/v1/messages` | `https://api.anthropic.com` | `https://api.anthropic.com/v1/messages` |
| Gemini | `/v1beta/models/:model:generateContent` | `https://generativelanguage.googleapis.com` | `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent` |

> 💡 **提示**：Base URL 无需包含 `/v1`、`/v1beta` 或具体的 API 端点路径，程序会自动处理。

---

### 📁 分组管理

分组用于将多个渠道聚合为一个统一的对外模型名称。

**核心概念：**

- **分组名称** 即程序对外暴露的模型名称
- 调用 API 时，将请求中的 `model` 参数设置为分组名称即可

> 💡 **示例**：创建分组名称为 `gpt-4o`，将多个供应商的 GPT-4o 渠道加入该分组，即可通过统一的 `model: gpt-4o` 访问所有渠道。
>
> 💡 **小技巧**：在渠道页，点击渠道右上角菜单中的「加入分组」，可把该渠道下的所有模型一次性加入到已有分组中。

---

### 💰 价格管理

管理系统中的模型价格信息。

**数据来源：**

- 系统会定期从 [models.dev](https://github.com/sst/models.dev) 同步更新模型价格数据
- 当创建渠道时，若渠道包含的模型不在 models.dev 中，系统会自动在此页面创建该模型的价格信息,所以此页面显示的是没有从上游获取到价格的模型，用户可以手动设置价格
- 也支持手动创建 models.dev 中已存在的模型，用于自定义价格

**价格优先级：**

| 优先级 | 来源 | 说明 |
|:------:|------|------|
| 🥇 高 | 本页面 | 用户在价格管理页面设置的价格 |
| 🥈 低 | models.dev | 自动同步的默认价格 |

> 💡 **提示**：如需覆盖某个模型的默认价格，只需在价格管理页面为其设置自定义价格即可。

---

### ⚙️ 设置

系统全局配置项。

**统计保存周期（分钟）：**

由于程序涉及大量统计项目，若每次请求都直接写入数据库会影响读写性能。因此程序采用以下策略：

- 统计数据先保存在 **内存** 中
- 按设定的周期 **定期批量写入** 数据库

> ⚠️ **重要提示**：退出程序时，请使用正常的关闭方式（如 `Ctrl+C` 或发送 `SIGTERM` 信号），以确保内存中的统计数据能正确写入数据库。**请勿使用 `kill -9` 等强制终止方式**，否则可能导致统计数据丢失。

## 🔌 客户端接入

> 💡 **提示**：本分支对外暴露的 API 与上游保持一致 (`OpenAI Chat / Responses / Anthropic / Images`)，
> 新增的图片生成/编辑接口透传路径为 `/v1/images/{generations,edits}`。

### OpenAI SDK

```python
from openai import OpenAI
import os

client = OpenAI(   
    base_url="http://127.0.0.1:18874/v1",
    api_key="sk-octopus-P48ROljwJmWBYVARjwQM8Nkiezlg7WOrXXOWDYY8TI5p9Mzg", 
)
completion = client.chat.completions.create(
    model="octopus-openai",  # 填写正确的分组名称
    messages = [
        {"role": "user", "content": "Hello"},
    ],
)
print(completion.choices[0].message.content)
```

### 🎨 图片生成 / 编辑 ( Images )

透传 `/v1/images/generations` 与 `/v1/images/edits` 两个端点，目前仅 OpenAI 协议渠道参与选路：

```python
from openai import OpenAI

client = OpenAI(base_url="http://127.0.0.1:18874/v1", api_key="sk-octopus-...")
resp = client.images.generate(prompt="一只戴墨镜的猫", n=1, size="1024x1024")
print(resp.data[0].url)
```

### Claude Code

编辑 `~/.claude/settings.json`

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:18874",
    "ANTHROPIC_AUTH_TOKEN": "sk-octopus-",
    "API_TIMEOUT_MS": "3000000",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "ANTHROPIC_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_SMALL_FAST_MODEL": "octopus-haiku-4-5",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "octopus-haiku-4-5"
  }
}
```

### Codex

编辑 `~/.codex/config.toml`

```toml
model = "gpt-5.6-sol"
model_reasoning_effort = "xhigh"
model_provider = "octopus"
preferred_auth_method = "apikey"

[model_providers.octopus]
base_url = "http://127.0.0.1:18874/v1"
name = "octopus"
supports_websockets = false
requires_openai_auth = true
wire_api = "responses"
experimental_bearer_token = "sk-octopus-"
```
编辑 `~/.codex/auth.json`

```json
{
  "OPENAI_API_KEY": ""
}
```


---

## 🤝 致谢

- 🙏 [looplj/axonhub](https://github.com/looplj/axonhub) - 本项目的 LLM API 适配模块直接源自该仓库的实现
- 📊 [sst/models.dev](https://github.com/sst/models.dev) - AI 模型数据库，提供模型价格数据
- 🇨🇳 [AtomGit](https://atomgit.com/bestruirui/octopus) - 国内代码托管
- 💬 [Linux.do](https://linux.do/)

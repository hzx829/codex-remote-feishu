# Codex Profile 与 OAuth 隔离设计

> Type: `draft`
> Updated: `2026-07-31`
> Summary: 统一 Codex/Claude 的用户可见 Profile 语义，设计 Codex OAuth 只读 Profile、API Profile 进程隔离、跨 Profile 会话恢复以及 Web/飞书管理交互。

## 1. 文档定位

本文定义 Codex Profile 的目标产品合同和技术实现边界。它取代旧的 `Web Codex Provider 管理设计`，并补齐当前实现尚未闭环的三部分：

1. 用户侧统一使用 `Profile`，不再暴露 `Provider` 作为产品概念。
2. 检测并保留 Codex 原生 ChatGPT OAuth 登录，将其投影为不可编辑、不可删除的只读 Profile。
3. Profile 切换必须真正作用于 URL、认证、模型和推理强度，并能安全恢复已有会话。

本文是方案草案，不表示对应代码已经实现。当前代码中的 `/codexprovider` 和 `Codex Provider` 仍属于待迁移现状。

## 2. 背景与已确认问题

当前仓库已经具备 Codex 自定义 Provider 的主要骨架：

- Web 管理页可保存名称、Base URL、API Key、模型和推理强度。
- daemon/wrapper 可通过 Codex `-c` 参数和子进程环境变量投影自定义 Provider。
- surface 和 managed instance 已携带 `CodexProviderID`，不同 Provider 不会被直接当作同一个启动合同复用。
- 飞书 `/codexprovider` 已支持选择后重启当前工作区。

但当前实现存在根因级缺口：

1. Remote 生成的 `thread/resume` 没有显式携带目标 `modelProvider`。Codex 会恢复会话持久化的旧 Provider，表现为切换后仍访问旧端点，或因旧 Provider 未在新进程注册而恢复失败。
2. orchestrator 只物化 Provider ID/名称，不知道 Profile 模型与推理默认值。当前 prompt 冻结路径可能把产品默认模型再次下发，覆盖 Profile 中配置的模型。
3. Provider ID 没有修订号。用户编辑同一个 Provider 后，旧进程仍可能因 ID 相同而被误判为兼容。
4. 当前 `codex_provider_env.go` 仍只按旧 `[profiles.<name>]` 格式解析 Codex 原生 Profile；上游当前 `main` 已改为 `$CODEX_HOME/<name>.config.toml`，但不能在未验证 release tag 的情况下硬编码版本分界。
5. 当前产品没有区分 Codex 原生 OAuth 凭据和本系统管理的 API Key，无法向用户明确保证“自定义配置不会覆盖 OAuth”。

## 3. 目标与非目标

### 3.1 目标

- Web 和飞书统一使用 `Claude Profile`、`Codex Profile`。
- Codex Profile 可管理与 Claude Profile 对齐的连接级字段：名称、端点、凭据、主模型、辅助模型和推理强度。
- 现有 ChatGPT OAuth 登录被发现后，形成一个只读 Codex Profile，可在机器人菜单中选择。
- 自定义 API Profile 不读取、不更新、不登出 Codex 原生 OAuth 凭据。
- 多个 managed instance 可同时使用不同 Profile，进程环境和启动合同彼此隔离。
- 切换 Profile 后可继续当前会话；目标 URL、认证、模型和推理强度必须与目标 Profile 一致。
- 不修改用户原有 `~/.codex/config.toml`、原生 Profile 文件或 OAuth 凭据。

### 3.2 非目标

- 不把 Codex Remote 变成多 OAuth 账号凭据保险库。
- 不复制或导出 OAuth access token、refresh token、`auth.json` 或 keyring 内容。
- 不开放任意 Codex 原生 Profile 字段，例如 MCP、hooks、sandbox、feature flags。
- 不支持运行中 turn 的热切换；Profile 切换只在当前工作空闲后通过重启/重连生效。
- 不允许 Web 创建 OAuth Profile。OAuth Profile 只能来自 Codex 原生登录探测。
- 不为每个 Profile 创建一套完整 `CODEX_HOME`。

## 4. 方案选择

### 4.1 方案 A：每个 Profile 独立 `CODEX_HOME`

优点是认证和配置天然隔离。缺点是 Codex 会话、SQLite 状态、skills、缓存和 keyring 存储键都与 `CODEX_HOME` 相关；若复制或软链其中一部分，会依赖上游私有目录结构，并容易产生同一会话多写者问题。

结论：不采用。

### 4.2 方案 B：复制 OAuth 凭据到本系统 Profile

这种方式理论上可保留多个 OAuth 身份，但新版 Codex 可能使用系统 keyring，外部无法可靠复制；复制 refresh token 也会扩大敏感凭据的存储面和刷新竞争。

结论：不采用。

### 4.3 方案 C：共享会话 Home，按进程隔离认证

OAuth Profile 继续只读使用 Codex 原生凭据存储。自定义 API Profile 仍共享用户 `CODEX_HOME` 中的会话和配置读取能力，但启动时强制使用进程内临时认证存储，并只注入当前 Profile 的 API Key。

结论：采用。它既不复制 OAuth，又能保留跨 Profile 的会话可见性。

## 5. 用户可见合同

### 5.1 最终用户

管理当前机器 AI 连接配置，并从飞书机器人为当前工作选择连接身份的用户。

### 5.2 当前任务

- 在 Web 中查看、创建和编辑 Codex Profile。
- 识别哪个 Profile 由 Codex 登录管理，哪个 Profile 由本系统管理。
- 在飞书中为当前机器人/工作区选择 Profile。

### 5.3 允许展示

- Profile 名称和类型：`ChatGPT 登录`、`本机默认`、`API`。
- 端点地址、模型、推理强度。
- API Key 是否已保存，不展示具体值。
- OAuth 登录是否被检测到、最近一次检测结果、脱敏账号提示和套餐类型（上游可用时）。
- 当前选择、切换中、切换成功、不可用原因。

### 5.4 禁止展示

- OAuth token、refresh token、完整 `auth.json`、keyring 键和值。
- API Key 明文。
- `model_provider`、`env_key`、`cli_auth_credentials_store`、`CODEX_HOME` 等实现字段。
- 内部 Profile 修订号、实例 ID、启动参数和协议 payload。

## 6. Profile 产品模型

### 6.1 Profile 类型

Codex Profile catalog 是以下三类记录的并集：

| 类型 | 来源 | 可编辑 | 可删除 | 启动语义 |
| --- | --- | --- | --- | --- |
| `native` | 用户现有 Codex 配置 | 否 | 否 | 不覆盖原生 Provider/认证，完整跟随本机默认 |
| `oauth` | Codex 原生 ChatGPT 登录探测 | 否 | 否 | 强制使用内建 OpenAI Provider 和原生 OAuth 存储 |
| `api` | Web 创建或旧 Provider 迁移 | 是 | 是 | 使用受管 Base URL/API Key/模型配置，隔离原生认证 |

`native` Profile 固定存在，展示名为 `本机默认`。检测到 OAuth 后，额外保留固定 ID 的 `oauth` Profile，展示名为 `ChatGPT 登录`。即使本机默认当前也使用同一 OAuth，两者也不能合并：前者会继续跟随用户的 Provider、环境和配置变化，后者必须锁定内建 OpenAI Provider 与 Codex 原生持久凭据，并清除可能抢占认证的外部环境变量。

Web 必须用副文案解释这两个入口的差别，不得只展示两个看似等价的名称：

- `ChatGPT 登录`：固定使用 Codex 当前登录账号。
- `本机默认`：完整跟随这台机器现有 Codex 配置。

### 6.2 数据结构边界

```go
type CodexProfileKind string

const (
    CodexProfileNative CodexProfileKind = "native"
    CodexProfileOAuth  CodexProfileKind = "oauth"
    CodexProfileAPI    CodexProfileKind = "api"
)

// 只用于 daemon 私有配置存储，禁止直接作为 HTTP 或状态机 DTO。
type CodexAPIProfileSecretConfig struct {
    ID              string           `json:"id,omitempty"`
    Revision        uint64           `json:"revision,omitempty"`
    Kind            CodexProfileKind `json:"kind,omitempty"`
    Name            string           `json:"name,omitempty"`
    BaseURL         string           `json:"baseURL,omitempty"`
    APIKey          string           `json:"apiKey,omitempty"`
    Model           string           `json:"model,omitempty"`
    ReviewModel     string           `json:"reviewModel,omitempty"`
    ReasoningEffort string           `json:"reasoningEffort,omitempty"`
}

// Web、orchestrator 和飞书只接触这个脱敏投影。
type CodexProfileSummary struct {
    ID              string
    Revision        uint64
    Kind            CodexProfileKind
    Name            string
    BaseURL         string
    Model           string
    ReviewModel     string
    ReasoningEffort string
    StatusCode      string
    Available       bool
    HasAPIKey       bool
    Editable        bool
    Deletable       bool
}
```

说明：

- `CodexAPIProfileSecretConfig` 和 `CodexProfileSummary` 必须是不同类型，禁止通过清空 `APIKey` 后复用 secret-bearing struct 作为响应。
- `Revision` 是单调递增的启动合同版本。名称、端点、Key、模型或推理强度变化都必须递增。
- OAuth/native Profile 不进入用户可写 `CodexAPIProfileSecretConfig`；API 返回时由 catalog projector 合成只读 summary。
- `ReviewModel` 对应 Codex 的 `review_model`，它是 Codex 最接近 Claude `smallModel` 的稳定辅助模型字段，但 Web 文案使用“审阅模型”，不伪装成完全相同的语义。
- API Key 仅存在 daemon 的 secret-bearing config 中。orchestrator、surface snapshot、instance hello 和 Web summary 都不得携带它。

### 6.3 只读 OAuth 描述符

OAuth 探测结果保存为不含凭据的状态记录：

```go
type CodexOAuthProfileState struct {
    ProfileID          string
    Revision           uint64
    Status             string
    AccountFingerprint string
    AccountHint        string
    PlanType           string
    LastCheckedAt      time.Time
    LastErrorCode      string
}
```

- 该描述符单独持久化；“保存 OAuth Profile”只保存固定 Profile 身份、修订和脱敏探测状态，不保存任何凭据副本。
- `Status` 是 `detected / missing / unknown` 三态：识别到 ChatGPT 登录、确认没有 ChatGPT 登录、探测本身未完成。`Available` 仅是 summary 中由 `detected` 派生的选择能力。
- `AccountFingerprint` 只能在 `account/read` 返回邮箱时由归一化邮箱计算带机器私钥的 HMAC；不能持久化普通邮箱哈希。上游不返回稳定账号 ID，因此该字段允许为空，不能用于恢复历史账号。
- `AccountHint` 最多保存脱敏邮箱；无法安全获得时留空。
- 一旦成功发现 OAuth Profile，后续 `missing` 或 `unknown` 都不会删除该 Profile；`unknown` 不能被误写成“已退出”。
- 不可用 OAuth Profile 不能静默回退到本机默认或 API Profile。
- 探测到账号身份变化，或状态在 `detected <-> missing` 间确认迁移时递增 OAuth `Revision`；短暂 `unknown` 和账号不变的 token 自然刷新不递增。

## 7. OAuth 探测与保护

### 7.1 探测方式

OAuth 探测使用短生命周期 Codex app-server probe，不直接解析 `auth.json`：

1. 使用用户原始 `CODEX_HOME`。
2. 清除 `OPENAI_API_KEY`、`CODEX_API_KEY`、`CODEX_ACCESS_TOKEN` 和本系统 Profile Key，避免环境认证遮住持久 OAuth。
3. 临时覆盖 `model_provider="openai"`，避免本机默认自定义 Provider 隐藏已有 OAuth。
4. 完成 app-server initialize。
5. 调用 `account/read`，固定 `refreshToken=false`。
6. 只接受 `account.type=chatgpt` 作为 OAuth Profile 证据。
7. 解析完成后立即退出 probe，不保留长期进程。

选择 app-server 协议而不是文件探测的原因：Codex 可将凭据存放在 file、keyring 或 auto backend；`account/read` 是能覆盖这些存储方式且不暴露 token 的官方边界。`refreshToken=false` 只证明“当前持久凭据可被 Codex 识别为 ChatGPT 登录”，不发起联网鉴权；真正的 token 有效性仍由目标实例首次请求或 Codex 自己的刷新结果确认，Web 不得把探测时间文案写成“账号已联网验证”。

### 7.2 探测时机

- daemon 启动后异步执行一次，不阻塞主服务就绪。
- Web 打开 Codex Profile 区时可显式刷新。
- 选择 OAuth Profile 启动实例前必须重新做轻量 preflight。
- 收到当前 OAuth 实例的 `account/updated` 时更新可用状态。

探测必须有独立超时和单飞控制。相同输入下的失败不能进入无限 retry；后台只在 daemon 重启、用户刷新、选择启动或 auth 事件变化时重新探测。

状态归类必须稳定：`account/read` 成功且 `account.type=chatgpt` 为 `detected`；成功但账号为空或不是 ChatGPT 为 `missing`；进程、协议、超时或配置读取失败为 `unknown`。选择 `missing/unknown` Profile 时执行一次显式 preflight，仍失败则保留对应结构化原因，不启动错误实例。

### 7.3 OAuth 只读保证

- Web 的 update/delete API 对 `native`、`oauth` 固定返回只读错误。
- OAuth Profile ID 是保留值，创建/导入 API Profile 时不能占用。
- 本系统不调用 `account/login/start` 或 `account/logout` 来管理该 Profile。
- OAuth token 刷新仍由 Codex 原生 AuthManager 完成，本系统不接管刷新令牌。
- 用户在本机执行 `codex login/logout` 属于外部变更；下次 probe 更新 Profile 可用状态，而不是尝试恢复旧 token。

这里的“保存/隔离”含义是：本系统保存只读 Profile 身份和选择状态，并保证 API Profile 不读取或覆盖原生 OAuth；它不承诺保存多个历史 OAuth 账号。

## 8. 运行时投影与认证隔离

### 8.1 单一解析边界

daemon 提供唯一解析函数：

```go
ResolveCodexProfileRuntime(profileID string) (CodexProfileRuntimeProjection, error)
```

投影分为两层：

- 公共启动合同：Profile ID、Revision、Kind、实际 Model Provider ID、模型、推理强度。
- 私有启动材料：CLI overrides 和 secret-bearing child env。

私有启动材料不能进入 orchestrator state、日志、错误 details 或 API response。

### 8.2 启动投影

| Profile | CLI 投影 | 认证环境 |
| --- | --- | --- |
| `native` | 不覆盖本机默认配置；解析出实际 Provider ID 供协议层使用 | 保留用户原生认证环境/存储 |
| `oauth` | `model_provider="openai"` | 清除外部认证环境后，使用 Codex 原生持久认证存储 |
| `api` | `model_provider`、`model_providers.*`、`model`、`review_model`、`model_reasoning_effort`、`cli_auth_credentials_store="ephemeral"` | 移除原生 OAuth/API 认证环境，只注入当前 Profile 专用 Key env |

API Profile 的用户可见 Profile ID 不能直接作为 Codex `model_provider`。resolver 必须生成带产品命名空间的内部 `CodexModelProviderID`，并确认它不与当前有效配置中的 Provider ID 冲突；内部 Provider 名称使用固定安全名称，不能把用户可编辑名称直接传给上游的 `OpenAI` 特殊判断。

API Profile 必须显式投影：

```text
-c model_provider="<internal-provider-id>"
-c model_providers.<internal-provider-id>.name="Codex Remote API"
-c model_providers.<internal-provider-id>.base_url="<base-url>"
-c model_providers.<internal-provider-id>.env_key="CODEX_REMOTE_CODEX_PROFILE_API_KEY"
-c model_providers.<internal-provider-id>.requires_openai_auth=false
-c cli_auth_credentials_store="ephemeral"
```

resolver 最终必须追加完整的目标模型配置：

```text
-c model="<model>"
-c review_model="<review-model>"
-c model_reasoning_effort="<effort>"
```

### 8.3 模型默认值闭包

只覆盖 `model_provider` 不能形成隔离。Codex 配置合并对 `model` / `review_model` / `model_reasoning_effort` 采用“override 优先，否则继承 base config”，且空字符串不是清除值；因此 OAuth/API Profile 都不能让未填写字段自然落回用户本机配置。

非 native Profile 在进入 instance contract 前必须得到完整的有效值：

1. 主模型优先使用 API Profile 显式值；没有显式值时，从使用目标 Provider/认证启动的短生命周期 probe 调用 `model/list`，选择 `isDefault=true` 的模型。
2. 推理强度优先使用 API Profile 显式值；没有显式值时，使用有效主模型对应的 `defaultReasoningEffort`。
3. 审阅模型优先使用 API Profile 显式值；没有显式值时，明确回退到本 Profile 的有效主模型，而不是继承 base config。
4. OAuth Profile 是只读项，三项都由同一次 OAuth/model probe 解析；probe 结果只缓存非敏感 catalog 字段，并按 OAuth Revision 失效。
5. 任一必需值无法解析或显式值不在已知支持范围时，返回 `profile_model_unresolved` / `profile_reasoning_unsupported`，不启动实例。

API Profile 的模型和推理强度在 Web 中仍可显示为“自动”；这里的“自动”严格指目标 Profile probe 的 catalog 默认值，不是“沿用本机默认”。实例启动、`thread/start`、`thread/resume` 和 restart restore 都使用这份已闭包投影。

### 8.4 环境清理

OAuth 和 API Profile 子进程都不能简单继承所有认证环境：

- `native` 保留原有环境，忠实跟随本机配置。
- `oauth` 移除 `OPENAI_API_KEY`、`CODEX_API_KEY`、`CODEX_ACCESS_TOKEN` 和本系统 Profile Key，保证只读取 Codex 原生持久 OAuth。
- `api` 移除同一组认证变量，再注入当前 Profile 专用 env key。

固定 env key 在多个实例之间可以复用，因为环境变量属于各自子进程；secret 值不能进入父进程全局环境。

### 8.5 版本门槛

`cli_auth_credentials_store="ephemeral"` 和 `thread/resume.modelProvider` 必须进入 Codex capability/version preflight。上游当前 `main` 已具备这些能力；若实际机器版本过旧或无法证明支持，API Profile 启动应明确失败并提示升级，不能降级为共享 OAuth 存储。

## 9. 实例合同与多实例隔离

### 9.1 选择作用域

Profile catalog 全局共享，但 Profile 选择不是全局开关：

- 飞书私聊中的切换与 Claude 现有行为对齐：更新当前 bot 的 Codex 默认值，并立即作用于当前 surface/workspace。
- 已经运行的其他 workspace/managed instance 保持原 Profile，不被批量重启；新建工作区继承该 bot 的当前默认值。
- workspace 恢复状态持久化自己的 Profile ID/Revision，因此 daemon 重启后仍恢复原合同。
- 这里的“临时切换”指不修改 Profile 内容、不写用户 Codex 配置，也不影响其他实例；切回原 Profile 即恢复原连接合同。

### 9.2 运行实例合同

实例 hello 和 desired contract 从单一 ID 扩展为：

```text
CodexProfileID
CodexProfileRevision
CodexModelProviderID
```

兼容判定必须同时满足：

- backend 为 Codex；
- Profile ID 相同；
- Profile Revision 相同；
- 实际 Model Provider ID 与目标投影一致。

Revision 来源必须覆盖三类 Profile：

- `api`：受管配置每次语义变更后单调递增。
- `oauth`：脱敏账号身份变化或 `detected <-> missing` 状态确认迁移时递增，不因普通 token 刷新或短暂 `unknown` 变化。
- `native`：daemon 根据配置源代次和脱敏后的有效 Provider/模型/启动参数推进 Revision；若 native 依赖同一 OAuth，再纳入 OAuth Revision。原始配置、凭据及其可离线猜测的无盐摘要都不能写入状态；无法证明合同未变化时生成新 Revision，允许多一次重启但不能错误复用。

因此：

- 两个实例可以在不同工作区使用不同 Profile。
- 同一个 API Profile 编辑后，旧 Revision 的实例不再被新请求复用。
- 已运行 turn 不因 Web 保存被强行中断；实例空闲后或下一次使用时按新 Revision 重建。
- 删除 API Profile 前，必须拒绝仍被 active/pending instance 或持久 surface selection 引用的 Profile，并提示先切换。

## 10. 会话恢复语义

### 10.1 根因修正

所有由 Remote 主动生成的 `thread/resume` 都必须携带目标运行时的 `modelProvider`，包括：

- 远程 prompt 恢复已有线程；
- compact 前隐式恢复；
- child restart restore；
- Profile 切换后的 exact-thread continuation。

原因是 Codex 在未显式覆盖时会优先恢复线程持久化的旧 Provider。

### 10.2 模型与推理强度

Codex 规定：只要 resume 显式传入 `modelProvider`，持久化的模型和推理强度 fallback 也会一起关闭。因此不能只补一个字段，还必须定义恢复策略：

1. 同 Profile 普通重启：显式传目标 `modelProvider`，同时携带已观测线程模型/推理强度，保持用户当前会话设置。
2. 切换到其他 Profile：显式传目标 `modelProvider`，使用目标 Profile 默认模型/推理强度。
3. 目标 Profile 在当前 workspace 存在用户显式 `/model`、`/reasoning` 快照：该快照覆盖 Profile 的闭包默认值。
4. Profile 字段选择“自动”时：使用目标 probe 已闭包的 catalog 默认值，不交给 base config 解析，也不注入产品硬编码默认值。

这要求 thread catalog/translator 继续携带 `modelProvider` observed state，并让 resume command 明确区分 `preserve_thread_settings` 与 `apply_target_profile`。

### 10.3 Profile 级临时覆盖

- `/model`、`/reasoning` 仍是当前 workspace + Profile 下的显式运行时覆盖。
- 切换 Profile 前保存当前 Profile 快照；切换后恢复目标 Profile 快照，没有快照则使用 Profile 默认值。
- 覆盖 key 必须包含 `backend + workspaceKey + profileID`，不能跨 Profile 泄漏。
- 已入队 prompt 使用冻结值，不受后续 Profile 切换影响。

## 11. Profile 切换状态机

切换只允许在没有 running turn、dispatching request、queue item 和 workspace preparation 时开始。

1. 校验目标 Profile 存在且可用。
2. OAuth/API Profile 分别完成 auth preflight 和 runtime projection。
3. 保存当前 workspace + Profile 的显式 override 快照。
4. 记录目标 desired Profile ID/Revision。
5. 停止不兼容的 managed instance，但不操作用户手动启动的 VS Code/CLI 实例。
6. 使用目标 Profile 启动新 managed instance。
7. 恢复原来的 route intent：`unbound`、`new_thread_ready` 或 exact-thread。
8. exact-thread 使用 `apply_target_profile` resume 策略。
9. attach 成功后提交切换状态；失败则保留目标选择和结构化根因，不无限重试、不回退旧 Profile。

Profile 切换失败必须区分：

- OAuth 已退出或不可读取；
- API Profile 缺少 Key/端点；
- 目标模型或默认推理强度无法闭包；
- Codex 版本不支持隔离能力；
- 目标 Provider 无法注册；
- thread resume 被其他 app-server 占用；
- workspace/thread 已不存在。

## 12. WebUI 设计

### 12.1 信息架构

Admin 保持当前配置管理页面骨架，将两个区块统一命名为：

- `Claude Profile`
- `Codex Profile`

本轮不把两类 Profile 混进同一列表，也不新增营销式首页。两个区块继续复用现有列表 + 详情编辑模式，保证 backend 字段和错误不会交叉。

### 12.2 Codex Profile 列表

列表排序固定为：

1. `ChatGPT 登录`（曾成功探测到时）
2. `本机默认`
3. 用户 API Profile，按名称排序

每一行只显示：

- Profile 名称；
- 类型图标/短标签：`OAuth`、`本机`、`API`；
- 一行摘要：登录/配置状态、端点主机或主模型。

OAuth 不可用时保留在列表中并显示状态，不直接消失。列表不显示 Profile ID、配置路径和完整邮箱。

当 `本机默认` 当前也解析到同一个 ChatGPT 登录时，两项仍分别显示，但摘要必须分别写成“固定使用当前 ChatGPT 登录”和“跟随本机 Codex 配置”；不能用两个相同的“可用”摘要让用户猜差别。

### 12.3 OAuth/native 只读详情

只读详情使用与 Claude 内建 Profile 相同的详情容器，但不渲染禁用表单控件。内容包括：

- 名称；
- `由 Codex 登录管理` 或 `跟随本机 Codex 配置`；
- 当前登录/配置状态；
- 脱敏账号提示和套餐（可用时）；
- 当前默认模型/推理强度（可解析时）；
- `重新检测` 操作。

不显示保存、删除和“覆盖认证”入口。重新登录属于本机 Codex 任务，错误态只提供可执行提示，不在 Web 中收集 OAuth token。

### 12.4 API Profile 编辑

编辑字段与 Claude Profile 保持相同节奏：

1. 名称，必填；
2. 端点地址，必填；必须是绝对 `http/https` URL，不允许 userinfo、query 或 fragment，本地地址可以使用 `http`；
3. API Key，创建时必填，更新时留空表示保留；
4. 主模型，可选，空值显示为“自动（目标 Profile 默认）”；
5. 审阅模型，可选，空值明确使用同一 Profile 的有效主模型；
6. 推理强度，可选，空值显示为“自动（目标模型默认）”，候选来自当前支持的 Codex 配置枚举。

API Key 只显示 `已保存` 状态，不回填。保存成功后递增 Revision；若有旧实例，反馈为“新配置将在下次使用该 Profile 时生效”，不承诺中断当前任务。

### 12.5 桌面与移动端

- 桌面保持左侧列表、右侧详情，主操作固定在详情底部。
- 移动端先显示单列列表；进入详情后使用页面级返回，不把双栏压成窄列。
- OAuth/native 详情和 API 编辑共用同一内容宽度，切换类型时页面不横向跳动。
- 错误只进入详情页现有 notice 槽位，不新增全页错误堆栈。

### 12.6 Web 状态

必须覆盖：加载中、探测中、已检测到 OAuth、OAuth 已退出、暂时无法检测、从未发现 OAuth、Profile 保存中、保存成功、删除冲突、版本过旧、读取配置失败。

Web API summary 使用结构化能力字段，例如 `editable`、`deletable`、`authKind`、`available`、`hasApiKey`；前端不能根据名称猜测只读状态。

## 13. 飞书菜单与命令

### 13.1 命名

- canonical slash：`/codexprofile`
- 菜单名称：`切换 Codex Profile`
- Claude 对应文案同步为 `Claude Profile`
- `/codexprovider` 保留为 help/menu 均隐藏的兼容 alias；功能首次发布后至少保留一个完整 minor 发布周期，再在后续 minor 删除

机器人菜单只负责选择，不提供 Profile 创建、编辑、删除和 OAuth 登录。

与现有 Claude Profile 一致，切换入口只在机器人私聊中开放；群聊菜单隐藏，手输命令时提示到私聊修改 bot 默认值，避免群成员改写共享 bot 能力设置。

### 13.2 卡片交互

`/codexprofile` 继续使用现有 config-flow 参数卡：

- bare open 和菜单内进入：`keep`
- 提交切换：同卡 patch 成成功/错误终态
- workspace 重启和恢复属于后台业务执行，不让菜单 launcher 长期承载过程状态
- stamped 卡保留现有返回菜单 footer；旧 lifecycle 卡点击必须拒绝

下拉项展示 Profile 名称和简短类型，不展示端点、邮箱和内部 ID。OAuth 不可用 Profile 可继续显示，但选择后必须同卡说明需要先在本机恢复登录。

### 13.3 卡片预算

Profile 参数卡仍是单表单、单下拉、单 notice、单 footer，不聚合诊断详情。catalog 的产品级上限固定为 50 个 API Profile，Profile 名称最多 64 个 Unicode code points 且不能包含换行/控制字符；加上只读项后仍需在序列化测试中验证真实 callback/message envelope 低于 30 KB 和 200 elements。超过上限在创建 API 时拒绝，不在卡片内临时截断导致 Profile 不可选择。

## 14. API、存储与迁移

### 14.1 API

新 canonical API：

```text
GET    /api/admin/codex/profiles
POST   /api/admin/codex/profiles
PUT    /api/admin/codex/profiles/{id}
DELETE /api/admin/codex/profiles/{id}
POST   /api/admin/codex/profiles/oauth/refresh
```

所有 list/write response 都使用 redacted summary。对 native/oauth 执行 PUT/DELETE 返回稳定只读错误码。

### 14.2 配置迁移

- `codex.providers[]` 一次性迁移为 `codex.profiles[]`，每条记录 `kind=api`，ID 保持不变，初始 Revision 为 1。
- surface、bot capability、workspace defaults 和 resume state 中的 `CodexProviderID` 迁移为 `CodexProfileID`。
- API Profile secret config 继续进入权限受限的 app config；OAuth 只读描述符进入独立 runtime state store；两者不能写入同一用户可编辑数组。
- 读取期允许旧字段作为迁移来源；新写入只写 Profile 字段，不能永久双写。
- `/codexprovider` 和旧 admin API 只作为 transport compatibility，不继续作为内部 SSOT；两者在功能首次发布后至少保留一个完整 minor 发布周期。
- 旧设计文档移入 `docs/obsoleted/`，本文成为新方案入口。

### 14.3 原生 Codex Profile 兼容

运行时 resolver 必须理解两代 Codex 配置形态：

- 旧版 `[profiles.<name>]`；
- 上游当前 `main` 使用的 `$CODEX_HOME/<name>.config.toml`。

该分界必须通过 capability/fixture 判断，不能依赖未经证明的版本号。兼容读取只用于解析用户现有 native 启动参数和实际 Provider ID，不把任意原生 Profile 自动导入可编辑 Web catalog。

## 15. 安全与诊断

- 配置文件权限继续使用仅当前用户可读写模式。
- 日志只记录 Profile ID/Revision/Kind、内部 Provider ID 和错误码，不记录 URL query、Key、token、完整账号。
- app-server OAuth probe 不记录原始 response。
- Profile runtime projection 的 secret env 与公共合同使用不同类型，避免误序列化。
- 认证失败必须保留底层错误分类，但用户主文案不暴露 token 或内部路径。
- 相同 Profile 启动失败是一次恢复 episode；只有用户重试、Profile 修改、OAuth 状态变化或目标变化才重新执行。

## 16. 实施分段

1. 建立 Profile 数据模型、Revision 和旧 Provider 迁移，不改变用户交互。
2. 实现 OAuth probe、只读 catalog 与 API Profile ephemeral auth 隔离。
3. 扩展 instance contract 和 translator resume config，修复跨 Profile exact-thread 恢复。
4. 修正 Profile 默认模型/推理强度与显式 override 的优先级及 workspace+Profile 快照。
5. 迁移 WebUI、飞书命令和用户可见文案到 Profile。
6. 删除内部旧 Provider SSOT 和到期兼容入口，同步 canonical 状态机文档。

阶段只表示执行顺序，不是默认停点。完整交付必须包含迁移、测试、文档和兼容清理。

## 17. 测试与验收

### 17.1 配置与安全

- 旧 Provider 配置无损迁移，API Key 不进入 summary/log/state。
- OAuth file、keyring、auto 三种存储由 app-server probe 正确识别，probe 环境中的认证变量不会遮住持久登录。
- OAuth Profile child env 不含外部 API/token auth env；API Profile child env 也不含这些变量，只含当前 Profile Key。
- probe 超时得到 `unknown`，成功读到空账号得到 `missing`，两者不会互相覆盖语义。
- native/oauth PUT/DELETE 均由后端拒绝。
- API Profile 编辑后 Revision 递增，旧实例不再兼容。

### 17.2 运行时

- OAuth、API A、API B 可分别启动独立实例。
- API A 与 API B 使用同名 env key 时值仍按进程隔离。
- 在 base config 预置错误模型、审阅模型和推理强度后，OAuth/API Profile 的闭包结果仍只来自目标 Profile/catalog。
- default -> API、OAuth -> API、API A -> API B、API -> OAuth 均使用目标 Provider。
- Profile 切换恢复已有 thread 时，`thread/resume.modelProvider` 必须等于目标投影。
- 同 Profile 重启保留线程模型/推理；跨 Profile 切换使用目标 Profile 默认或目标快照。
- OAuth 失效、版本过旧、Provider 未注册和 thread busy 分别产生不同稳定错误。

### 17.3 WebUI

- 桌面和移动端都能完成查看、创建、编辑、删除。
- OAuth/native 详情无可编辑控件，API Profile 有完整字段。
- API Key 永不回填；保存、冲突和探测失败只进入固定 notice 槽位。
- 长名称、最长模型名和 50 个 Profile 不溢出布局。

### 17.4 飞书

- `/codexprofile` bare open、菜单 handoff、提交、返回和旧卡拒绝均有回归测试。
- 新菜单项继续遵守 `keep / enter_owner / enter_terminal` 统一合同；本设计保持 config-flow `keep`。
- 群聊菜单隐藏该入口，群聊手输稳定提示到私聊修改。
- 50 个 API Profile 加只读项时卡片仍在 transport/element 预算内。
- `/codexprovider` 隐藏 alias 在迁移期可用，但不出现在 help/menu。

## 18. 完成标准

以下条件全部满足才算功能完成：

1. 用户可见位置只使用 `Profile`，不再显示 `Provider`。
2. OAuth 被投影为可选择、不可编辑、不可删除的 Profile。
3. API Profile 进程无法读取或覆盖原生 OAuth 存储。
4. 多实例不同 Profile 不会被错误复用。
5. Profile 编辑后的旧实例不会继续冒充新配置。
6. 跨 Profile 恢复已有会话时，端点、认证、模型和推理强度全部来自目标 Profile 或其显式快照。
7. 用户原有 Codex 配置、原生 Profile 文件和 OAuth 凭据没有被写入或迁移。
8. Web/飞书、配置迁移、协议帧、敏感信息脱敏和状态机文档均完成验证。

## 19. 实现参考

- `internal/config/codex_providers.go`
- `internal/config/codex_provider_env.go`
- `internal/app/daemon/app_headless_codex_provider.go`
- `internal/app/daemon/admin_codex_providers.go`
- `internal/core/state/codex_provider.go`
- `internal/core/orchestrator/service_codex_provider_command.go`
- `internal/core/orchestrator/service_surface_contract_compatibility.go`
- `internal/adapter/codex/translator_commands.go`
- `internal/adapter/codex/translator_restart_restore.go`
- `web/src/routes/admin/CodexProviderSection.tsx`
- `web/src/routes/admin/ClaudeProfileSection.tsx`
- `docs/general/config-state-storage-guidelines.md`
- `docs/general/feishu-menu-card-usage-guidelines.md`
- `docs/general/feishu-card-ui-state-machine.md`
- `docs/general/remote-surface-state-machine.md`

## 20. 上游调研依据

本设计在 OpenAI Codex `f0c30e528a54bdf0fa9a4d52ff74b34383434811`（`2026-07-31` 的 `origin/main`）上复核了以下边界；实施时仍需用项目支持的最低 Codex 版本做 capability fixture，不能只按最新源码编译假设：

- `codex-rs/app-server-protocol/src/protocol/v2/account.rs`：`account/read` 返回 `apiKey` / `chatgpt` 等结构化账号类型，ChatGPT 只暴露可选邮箱和套餐，不暴露 token 或稳定账号 ID。
- `codex-rs/app-server/src/request_processors/account_processor.rs`：`refreshToken=false` 不触发主动刷新。
- `codex-rs/config/src/types.rs`、`codex-rs/login/src/auth/storage.rs`、`codex-rs/login/src/auth/manager.rs`：`ephemeral` 是进程内存存储，且显式选择后不会回退 file/keyring/auto；认证环境仍有独立优先级，因此必须同时清理环境变量。
- `codex-rs/app-server-protocol/src/protocol/v2/thread.rs`：`thread/resume` 支持显式 `modelProvider`、`model` 并返回实际 `modelProvider`、模型和推理强度。
- `codex-rs/app-server/src/request_processors/thread_processor.rs` 及其测试：只要显式覆盖 model、model provider 或 reasoning，持久化模型/Provider/推理三项就整体停止回填，因此跨 Profile 恢复必须成组决定这些字段。
- `codex-rs/core/src/config/mod.rs`：模型类配置采用 override 优先、否则继承 base config 的合并方式，不能用空字符串表达“清除”。
- `codex-rs/app-server-protocol/src/protocol/v2/model.rs`：`model/list` 明确返回默认模型标记、默认推理强度和支持档位，可作为非 native Profile 默认值闭包的协议依据。
- `codex-rs/cli/src/lib.rs`：新版 `--profile` 从 `$CODEX_HOME/<name>.config.toml` 叠加配置；旧 `[profiles.<name>]` 只能作为兼容读取来源。

# Feishu Group On-Demand Resume Design

> Type: `draft`
> Updated: `2026-07-26`
> Summary: 记录 Feishu 群聊重启后不刷屏、但被 @ 时按群上下文恢复的产品边界、实现方向与当前恢复失败刷屏原因。

## 背景

当前 `surface resume state` 同时承担两类职责：

1. 持久化 surface 上下文：重启后记住私聊或群聊的 `ProductMode`、backend、workspace、thread、route、room context。
2. daemon 后台自动恢复：重启后在没有用户交互时，主动尝试恢复这些 surface，并把恢复成功或失败通知发回对应飞书窗口。

这两个职责对私聊基本合理，但对群聊不合理。历史群 surface 数量可能很多，daemon 启动或升级重启后不应该把所有历史群都当成当前前台窗口去自动拉起 headless、自动恢复、自动失败通知。

## 产品目标

目标行为：

1. 私聊保持现状：daemon 重启后可以后台自动恢复，并可在私聊里提示恢复成功或失败。
2. 群聊不做无人触发的后台恢复：daemon 启动、升级重启、tick 不主动给历史群拉起实例，也不主动向群里发恢复成功或失败。
3. 群聊保留上下文：群 surface 的 resume entry 仍持久化，room workspace binding 仍能 materialize。
4. 群聊被 @ 时按需恢复：用户在群里 @ 对应机器人并发送消息时，如果该群 surface 有可恢复的 workspace/thread，应在这次用户触发的入站动作里尝试恢复或启动。
5. 群聊主动恢复失败时只回复当前交互：失败提示可以发到群里，因为这是用户刚刚 @ 触发的结果，不是后台噪声。
6. 升级/重启生命周期通知只发私聊：群聊不接收“服务正在关闭/恢复”这类全局生命周期广播。

## 目标体验

daemon 重启后，历史群里不会出现恢复失败刷屏。

用户在群里 @ 机器人发送普通文本时：

1. 如果原 workspace 可恢复且没有被其他私聊或其他群占用，则启动或接管对应 backend，然后把这条消息排队发送。
2. 如果原 thread 可恢复，则优先恢复原 thread。
3. 如果原 thread 不可见但 workspace 可恢复，则回到该 workspace，并按当前 headless 语义新建或选择会话。
4. 如果 workspace 不存在、provider/profile 不可用、backend runtime 缺失，则对这次 @ 返回明确失败原因。
5. 如果同一个群内另一个机器人正在处理同 workspace，则沿用 room `ActiveLock`，提示等待，不并发启动两个实例。

用户没有 @ 时：

1. 不 materialize 新动作。
2. 不触发恢复。
3. 不发失败提示。

## 实现方向

### 1. 拆分 resume entry 的两个职责

保留 `surface resume state` 作为持久化上下文 SSOT，但不要直接把所有 entry 都放进同一个 background recovery queue。

建议拆出两个判定：

1. `entryAllowsBackgroundRecovery(entry)`：只允许私聊和非 Feishu legacy surface 进入 daemon 启动/tick 后台恢复队列。
2. `entryAllowsOnDemandRecovery(entry, action)`：允许 Feishu 群聊在被当前 bot @ 的入站动作里恢复。

群聊 entry 不应被删除，也不应丢失 resume target。

### 2. 增加群聊 on-demand resume 入口

位置应在 daemon 入站动作进入 `service.ApplySurfaceAction` 前后都可以，但要满足两个条件：

1. 在普通 `handleText` 返回 `not_attached` 之前完成恢复意图处理。
2. 恢复过程必须保留当前用户消息，恢复成功后这条消息要继续进入 queue，而不是要求用户再发一遍。

推荐路径：

1. daemon 收到 Feishu 群聊 @ 文本后，确认 action 已通过现有 mention gate。
2. 如果 surface 当前 detached 且 persisted resume entry 有 workspace/thread target，则进入 on-demand recovery。
3. 复用现有 `TryAutoResumeHeadlessSurface` / workspace continuation 解析能力，但以“当前 action 触发”为上下文。
4. 若恢复需要启动 headless，则创建 `PendingHeadless`，把当前消息作为等待恢复后的 continuation queue item 或 pending input。
5. 若恢复能立即 attach visible/managed instance，则继续正常 `ApplySurfaceAction` 处理当前文本。

### 3. 生命周期通知目标收口

`beginShutdownNotices()` 当前遍历 `service.Surfaces()`，这个集合包含持久化 materialized 的历史群 surface，不等价于“需要系统通知的窗口”。

应新增单一策略函数，例如 `daemonLifecycleNoticeAllowedForSurface(surface)`，规则：

1. Feishu 私聊允许。
2. Feishu 群聊不允许。
3. 非 Feishu surface 按现有行为保留，除非后续明确产品策略。

该策略只用于 daemon lifecycle notice，不用于用户主动命令结果。

## 当前事故分析

日志证据：

1. `2026-07-26 14:08:38` 出现 `headless kill requested: surface=feishu:Codex-5:chat:oc_1eb491c89f520efd0255b3717ed9e3a0 ...`。
2. 之后同一个群 surface 连续出现 `kind=notice`，间隔约 2-3 秒。
3. 这个 surface 是 Feishu 群聊 surface：`feishu:Codex-5:chat:oc_...`。

直接原因：

1. daemon 重启后从 `surface resume state` materialize 了历史群 surface。
2. 该群 surface 进入了后台恢复链路。
3. 后台恢复尝试拉起 headless 后，启动或恢复没有成功，进入 `PendingHeadless` 超时/失败路径。
4. `Tick()` 中的 `expirePendingHeadless` 发出 `headless_restore_start_timeout`，并触发 `DaemonCommandKillHeadless`。
5. 群聊被当作普通 surface 接收了这些恢复失败 notice，于是出现群里刷屏。

为什么之前“恢复失败原因和去重”没有挡住：

1. 之前的修正主要覆盖 `recordSurfaceResumeFailureLocked` 管理的 surface resume episode：即 `TryAutoResumeHeadlessSurface` 返回 `SurfaceResumeStatusFailed` 后，稳定失败根因、backoff、`LastNoticeCode` 这些状态不重复刷。
2. 这次日志里出现了 `headless kill requested`，对应的是已经进入 `PendingHeadless` 后的超时/kill 路径。
3. `expirePendingHeadless` 会直接生成 `headless_restore_start_timeout` notice；这不是单纯的 `NoticeForSurfaceResumeFailure` 出口。
4. 更关键的是，之前修正没有解决“群聊是否应该进入后台恢复队列”这个边界，所以即使某些失败能去重，群聊仍可能被 daemon 主动恢复并产生其它路径的系统通知。

结论：之前修正不是完全错误，但它解决的是“恢复 episode 内失败原因稳定和部分重复提示”，没有覆盖群聊产品边界，也没有把 `PendingHeadless` 超时 notice 统一纳入同一套后台恢复通知策略。因此这次行为说明修正范围不够深。

## 开工前需要确认

当前建议默认采用：

1. 群聊不进入 background recovery。
2. 群聊 @ 时进入 on-demand recovery。
3. 群聊 on-demand recovery 成功后继续处理本次消息。
4. 群聊 on-demand recovery 失败时只返回本次交互的失败原因。
5. 生命周期通知只发私聊。

尚需实现时细化：

1. 当前消息在 pending headless 期间如何作为 continuation 保存。
2. on-demand 恢复失败是否要写入后台 recovery backoff，避免同一个用户连续 @ 时无间隔重试。
3. 私聊和群聊是否共享同一套失败原因文案，但投递策略不同。
4. existing tests 中哪些 background recovery 用例需要拆成 private / group 两类。

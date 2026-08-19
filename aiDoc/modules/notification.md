# Notification 模块

## 职责与边界

`notification` 拥有通知意图、独立发送尝试、标准回执、补发、供应商契约门禁和送达异常待办。它不决定任务是否存在、是否完成或是否通过复核，也不发送真实短信。

默认运行时只启用不访问网络的 `DemoNotificationAdapter`。P2-03 增加 `ProviderContractAdapter`，用于验证签名、受理、异步回执、限流和费用边界；它只依赖 `ProviderGateway` port，仓库没有提供网络实现，也不接收手机号或通知正文。配置只允许 `DISABLED|CONTRACT_TEST`，不接受生产模式。

## 入口

- 后端模型：`server/model/notification/`
- 服务与 adapter：`server/service/notification/`
- 员工 API：`server/api/v1/notification/`
- 私有路由：`server/router/notification/`
- 迁移、菜单、权限和固定记录：`server/initialize/notification_seed.go`
- 前端 API：`web/src/api/sleep-care/deliveries.js`
- 页面：`web/src/view/sleep-care/deliveries/index.vue`

任务人工联系仍由 `carepath` 聚合拥有，通过 `POST /care/tasks/{id}/contact-records` 追加任务事件。

## 持久模型

- `NotificationRequest`：一个任务对应一个稳定通知意图。
- `NotificationAttempt`：一次独立发送；以请求 ID 和 `attemptNo` 唯一，带独立版本、时间戳和补发来源。
- `DeliveryEvent`：只追加的标准状态证据，以 attempt 和事件键去重。
- `NotificationDispatchReservation`：一次 provider attempt 的唯一额度与费用预留。
- `NotificationQuotaBucket`：按不可变策略版本、时间窗或 `Asia/Shanghai` 业务日原子累计。
- `NotificationProviderCallback`：只保存验签后的事件、nonce、payload 摘要和标准状态，不保存原始供应商标识。

失败或未知通过统一 `TodoItem` 写入 `DELIVERY_ISSUE`，其来源为逻辑通知请求。因此同一请求多次失败或未知只保留一个活动待办；后续 attempt 确认送达时才完成该待办。

## 状态不变量

```text
PENDING -> SUBMITTED_TO_PROVIDER -> ACCEPTED -> DELIVERED
                                         \---> FAILED
                                         \---> UNKNOWN
```

- `ACCEPTED` 只证明通道受理，`deliveredAt` 仍为空。
- 三个终态都不可再写入新回执；同一事件键重放返回原事实。
- `FAILED` 与 `UNKNOWN` 才可补发；补发创建新 attempt 并通过 `retryOfAttemptId` 指向旧 attempt。
- provider attempt 在创建事务内先完成门禁、最大尝试次数、限流和费用预留；任一拒绝都会回滚新 attempt，不伪造提交事实。
- 相同发送命令重放只返回已有 attempt，不再次调用 adapter；额度预留以 attempt 唯一，不能重复占用。
- 通知状态、客户端交互、任务执行、时效和复核彼此独立。
- 任何通知异常都不得取消、删除、重排或回滚 D1–D5。

## 接口与权限

- `GET /care/deliveries`：管家、医护和上级可读授权范围记录；列表内含版本与事件证据。
- `GET /care/notification-provider-readiness`：管家、医护和上级只读当前门禁、重试、限流和费用边界；不返回密钥。
- `POST /care/deliveries/{id}/resend`：仅当前责任管家；要求 `Idempotency-Key`、旧 attempt 的 `expectedVersion` 和原因。
- `POST /care/notification-provider-callbacks/{providerCode}`：公开回调入口，不使用员工 JWT；只有配置门禁、HMAC 验签、时间窗和 nonce 防重放全部通过后才写业务事实。
- `POST /care/tasks/{id}/contact-records`：当前责任管家或医护；要求幂等键、任务版本、渠道、结果和发生时间。

DataScope 负责部门过滤；业务访问策略再叠加当前有效责任关系。菜单与按钮只改善交互，不替代服务端授权。

## Provider 契约安全边界

- 请求和回调签名串固定为 `timestamp + "\n" + nonce + "\n" + rawBody`，算法为 HMAC-SHA256。
- 回调要求 Unix 秒时间戳、16–128 字符 nonce、最大 64 KiB JSON 和精确原始 body 验签；默认允许偏差 300 秒。
- callback event 与 nonce 分别按 provider 唯一；相同事件和相同 payload 安全重放，不同 payload 或复用 nonce 拒绝。
- provider message ID 只在内存中用于验签后定位，attempt、事件与回调表只保存 SHA-256 摘要。
- 全局访问日志遮蔽签名、nonce、provider message ID 和 event ID；回调路由在 AccessLog 读取前限制 body。
- `ACCEPTED` 只来自明确受理回执，异步终态时间不得早于受理时间。

## 重试、限流与费用

- 终态补发仍使用现有新 attempt 语义；provider 策略额外冻结 `maxAttemptsPerRequest`。
- 时间窗限流与日费用使用数据库条件更新消费 bucket，额度和费用在同一事务内预留；费用只使用整数最小单位，不使用浮点数。
- 同一策略 code/version 对应的 bucket limit 和时间边界不可原地漂移；需要修改时发布新版本。
- provider gateway 返回错误时，不写 `SUBMITTED_TO_PROVIDER` 或 `ACCEPTED`；已建立的 attempt 和预留保留为 `PENDING|RESERVED`，等待后续运行保障流程处理。
- 默认配置的 provider、模板、密钥和额度为空，网络发送与正式发送状态始终为关闭。

## 场景 B

固定记录使用客户 `20002`、计划 `23002`、任务 `24101`–`24105` 和 attempts `28101`/`28102`。第一次尝试为 `FAILED`，第二次补发为 `UNKNOWN`；D1 保持 `OPEN`，D2–D5 保持 `SCHEDULED`，活动异常待办只有一条。

场景 B 的请求、首次 attempt、失败回执、补发、未知回执和待办由同一外层事务初始化。任一步失败时不保留半成品 attempt；重试启动可幂等补齐整个场景。

## 测试边界

定向测试覆盖适配器序列、最小 payload、请求签名、回调篡改/过期/重放、非法跳转、时间倒退、终态保护、事件去重、补发幂等、最大尝试次数、原子限流、费用上限、版本冲突、责任范围、角色权限、唯一待办、任务不变和初始化幂等。阶段内不做页面点触，阶段二集中浏览器验收归 P2-08。

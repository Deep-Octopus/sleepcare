# Notification 模块

## 职责与边界

`notification` 拥有通知意图、独立发送尝试、标准回执、补发和送达异常待办。它不决定任务是否存在、是否完成或是否通过复核，也不发送正式短信。

阶段一只启用不访问网络的 `DemoNotificationAdapter`。适配器不接收手机号或通知正文，只根据固定结果输出标准状态事件；任何正式渠道、模板、签名、验签、限流或供应商密钥都留在后续门禁之后。

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
- 通知状态、客户端交互、任务执行、时效和复核彼此独立。
- 任何通知异常都不得取消、删除、重排或回滚 D1–D5。

## 接口与权限

- `GET /care/deliveries`：管家、医护和上级可读授权范围记录；列表内含版本与事件证据。
- `POST /care/deliveries/{id}/resend`：仅当前责任管家；要求 `Idempotency-Key`、旧 attempt 的 `expectedVersion` 和原因。
- `POST /care/tasks/{id}/contact-records`：当前责任管家或医护；要求幂等键、任务版本、渠道、结果和发生时间。

DataScope 负责部门过滤；业务访问策略再叠加当前有效责任关系。菜单与按钮只改善交互，不替代服务端授权。

## 场景 B

固定记录使用客户 `20002`、计划 `23002`、任务 `24101`–`24105` 和 attempts `28101`/`28102`。第一次尝试为 `FAILED`，第二次补发为 `UNKNOWN`；D1 保持 `OPEN`，D2–D5 保持 `SCHEDULED`，活动异常待办只有一条。

## 测试边界

定向测试覆盖适配器序列、非法跳转、终态保护、事件去重、补发幂等、版本冲突、责任范围、角色权限、唯一待办、任务不变和初始化幂等。阶段内不做页面点触，集中浏览器验收归 P1-11。

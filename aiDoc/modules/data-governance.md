# 数据治理门禁与生命周期请求

## 模块职责

P2-05 在 `careClient` 模块内提供数据治理准备状态和固定测试请求台账。它用于显式暴露真实数据、正式同意和数据处置的关闭状态，并为后续业务与合规评审保留稳定接口，不代表对应真实能力已经启用。

## 入口与依赖

- 后端模型：`server/model/careclient/data_lifecycle_request.go`
- 后端服务：`server/service/careclient/data_governance.go`
- 后端 API：`server/api/v1/careclient/care_client.go`
- 后端路由：`server/router/careclient/care_client.go`
- 运行配置：`server/config/care.go` 的 `care.data-governance`
- 前端入口：`web/src/view/sleep-care/clients/components/DataGovernancePanel.vue`
- 依赖现有康养用户、业务角色、Casbin、DataScope、命令回执和乐观锁，不创建新角色。

## 数据与状态

`care_data_lifecycle_requests` 是追加式请求台账。每条记录包含机构、康养用户、请求类型、请求时间、记录来源、事实说明、身份核验状态、治理模式、策略快照摘要和公共审计字段。

当前请求类型固定为：

- `ACCESS_COPY`
- `CORRECTION`
- `RESTRICTION`
- `ERASURE`

当前唯一状态是 `PENDING_POLICY`，身份核验状态固定为 `NOT_CONFIGURED`，`executionAllowed` 永远为 `false`。没有更新、状态转换、导出、删除、匿名化或批处理接口。

## 接口契约

| 方法 | 路径 | 契约 |
| --- | --- | --- |
| `GET` | `/care/data-governance-readiness` | 返回模式、用途、四类正式同意准备项、治理评审项和阻塞项 |
| `GET` | `/care/clients/{id}/data-lifecycle-requests` | 分页读取授权范围内固定测试用户的请求台账 |
| `POST` | `/care/clients/{id}/data-lifecycle-requests` | 在显式测试门禁下追加一条待政策请求 |

写请求要求 `Idempotency-Key`、`expectedVersion`、合法请求类型、带时区请求时间、`STAFF_RECORDED` 来源和事实说明。成功后增加康养用户版本；相同键与相同请求重放原结果，相同键与不同请求冲突。

## 权限与数据范围

- 三个接口只授权 `SUPERVISOR`。
- 后端继续叠加 JWT、改密保护、Casbin、DataScope、业务角色、机构范围和固定测试记录检查。
- `DeptId` 与 `CreatedBy` 由统一 DataScope 回调盖章，Service 不手写范围条件或公共审计字段。
- 普通管理员、管家、医护、内容管理员、跨机构身份、非测试记录和缺失身份均失败关闭。

## 配置与启用边界

`care.data-governance.mode` 只接受 `DISABLED|CONTRACT_TEST`。`CONTRACT_TEST` 还必须同时开启固定测试数据门禁；仓库默认配置为 `DISABLED`。

配置只保存非敏感版本号和评审布尔值，不保存正式同意正文、身份材料、保存期限、审批凭据或外部系统密钥。四类同意准备项为服务告知、隐私、通知和智能处理；即使评审项全部通过，当前实现仍返回：

- `realDataEnabled=false`
- `formalConsentEnabled=false`
- `lifecycleExecutionEnabled=false`

显式测试模式只控制是否可以记录待政策请求，不能开启真实数据或任何处置动作。

## 迁移与回退

迁移通过现有 GORM AutoMigrate 新增 `care_data_lifecycle_requests` 表，不修改或删除现有用户、授权记录和服务过程数据。回退代码不会自动删表；物理清理需要独立审批、备份和迁移方案。

## 必须保持的不变量

1. 未取得正式制度和责任人确认前，真实数据、正式同意和数据处置始终关闭。
2. 请求台账只追加，不原地修改或删除。
3. 请求事实不能被解释为已完成身份核验或已执行用户诉求。
4. 不增加真实短信、用户侧智能能力或医疗内容。
5. 前端门禁和按钮不可替代后端权限与配置校验。

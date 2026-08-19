# Supervision 模块边界

## 职责

`supervision` 为授权上级提供每日汇总、历史版本、复核队列、追加指导、直接介入、匿名服务评价和低分质量跟进能力。它读取 `careClient`、`carePath` 与 `caseWork` 已有事实形成投影，不修改答卷、规则命中、咨询互动或下级历史行动。

当前阶段只处理固定测试记录，不启用正式日报调度、真实通知、医疗内容或面向用户的 AI。

## 后端与前端入口

- Model：`server/model/supervision/`
- Service：`server/service/supervision/`
- API：`server/api/v1/supervision/`
- Router：`server/router/supervision/`
- 关注事项事务 seam：`server/service/casework/supervisor.go`
- 迁移和初始化：`server/initialize/gorm_biz.go`、`server/initialize/supervision_seed.go`
- 每日汇总页面：`web/src/view/sleep-care/daily-summaries/`
- 待复核页面：`web/src/view/sleep-care/review-queue/`
- 服务评价页面：`web/src/view/sleep-care/satisfaction/`
- 前端请求：`web/src/api/sleep-care/supervision.js`
- 评价前端请求：`web/src/api/sleep-care/satisfaction.js`

员工路由挂在系统统一 PrivateGroup 下，沿用 `JWT -> MustChangePwdGuard -> Casbin -> DataScope` 中间件链。Service 层继续透传请求 Context，并额外校验业务角色、管理部门和固定测试门禁。

## 数据模型

### DailySummaryVersion

历史汇总是追加式不可变版本，以 `organizationId + businessDate + version` 唯一。每条记录保存指标口径版本、生成时间、七项指标和重点事项 JSON 快照；相同业务日期的修正只能创建更高版本。

当前七项指标为：

- `servedClients`：业务日内存在到期任务的去重服务人数；
- `dueTasks`：业务日内应执行任务数；
- `submittedTasks`：统计截止时点前提交的任务数；
- `deliveryIssues`：截止时点仍存在的送达异常待办数；
- `openAttentionCases`：截止时点未关闭的关注事项数；
- `resolvedAttentionCases`：业务日内已解决事项数；
- `reviewRequired`：截止时点等待上级复核的事项数。

时区固定为 `Asia/Shanghai`。今日列表首项为请求时计算的 `REALTIME_PREVIEW`，`id=0` 且没有版本号；它不会写入历史表。历史项为 `VERSIONED_SNAPSHOT`，只能通过详情接口读取其冻结重点事项。

今日实时预览的截止点覆盖当前业务毫秒，因而与固定时钟恰好相同的已持久化提交、解决或关闭事实也会被纳入。历史快照仍使用自然日的半开区间，不改变历史口径。

### SupervisorGuidance

上级指导与介入均为追加事实，保存关联事项行动、动作类型、内容、操作者、后续责任医护、截止时间、动作前后事项版本及幂等请求摘要。表内记录不更新下级原行动，也不承担复核队列本身的状态存储。

复核状态由行动流投影：

- `PENDING`：最近一次请求尚无上级动作；
- `GUIDED`：最近动作是上级指导，事项仍等待复核；
- `INTERVENED`：最近动作是直接介入，事项回到处理中；
- `COMPLETED`：复核动作后已有后续处理或事项离开当前复核周期。

指导后再次由医护请求复核会开启新的 `PENDING` 周期，不覆盖上一轮记录。

### SatisfactionPolicy / Request / Response

- `SatisfactionPolicy` 是不可变策略版本，保存触发类型、生效时间、匿名方式、有效期和低分阈值。
- `SatisfactionRequest` 由一次咨询关闭事件投影，按 `sourceType + sourceEventId` 唯一，并冻结策略快照和系统授权关联。
- `SatisfactionResponse` 每个 request 只允许一条，保存 1–5 星、可选意见和提交幂等摘要。
- 客户端 DTO 只返回当前会话本人邀请；员工 DTO 采用 `STAFF_ANONYMOUS_SYSTEM_LINKED`，不返回用户、咨询或服务责任人关联。

### SatisfactionFollowUp / Action

低分 response 在同一事务内创建 `SatisfactionFollowUp`、统一活动待办与 outbox。跟进状态为 `OPEN → IN_REVIEW → RESOLVED`；接收与解决动作写入只追加的 `SatisfactionFollowUpAction`。解决命令必须确认单条评价的使用边界，并同步完成活动待办。

## 写事务与责任链

`SupervisionService` 持有指导记录、幂等检查和最外层事务；它通过 `CaseWorkService.ApplySupervisorAction` 在同一事务内追加 `CaseAction`、更新关注事项和活动待办，并写入 `SupervisorGuidanceAdded` outbox 事实。

- 指导或讨论安排：事项保持 `WAITING_SUPERVISOR`，活动待办保持开放，但责任医护和截止时间更新；
- 直接介入：事项进入 `HANDLING`，活动待办转交给明确的有效责任医护并更新截止时间；
- 两类动作均要求 `Idempotency-Key`、`expectedVersion`、非空内容、有效责任医护和未来截止时间；
- 相同键和相同请求重放原结果，相同键和不同请求拒绝，过期版本拒绝。

关注事项聚合先完成权限校验；随后读取规则命中和行动等子事实时，显式跳过子表自身的部门盖章，避免上级行动记录的部门归属阻断当前责任医护查看完整聚合历史。该跳过只用于已授权聚合的子事实，不允许按子表独立扩权查询。

咨询关闭评价使用反向 port 保持模块方向：`caseWork` 声明 `ConsultationClosedProjector`，组合入口注入 `SupervisionService`。关闭、互动、咨询待办、咨询 outbox、评价邀请和评价 outbox 共用同一事务。补偿扫描以关闭事件唯一键幂等补齐已有事实，不覆盖旧邀请。

## 权限

- 只有业务角色 `SUPERVISOR` 拥有员工督导/评价菜单、按钮和对应 Casbin 策略；客户端三个评价接口继续使用受限会话认证与同源保护；
- 上级只能读取其 DataScope 可见部门所归属机构的汇总、事项和指导事实；
- 跨机构、普通管理员、管家、医护及缺少业务身份的调用均失败关闭；
- 前端按钮仅控制入口可见性，不能替代后端授权与乐观锁校验。

## 迁移和生成边界

GORM 自动迁移包含 `daily_summary_versions`、`supervisor_guidance` 以及五张评价策略/邀请/响应/跟进/动作表。固定测试开关启用时，初始化幂等登记 API、菜单、按钮、角色策略、评价策略版本，并仅在缺少目标记录时生成一条固定历史快照。

当前没有对外的日报生成接口，也没有定时任务。正式业务日期归属、跨日转交、自动生成和运营口径仍受后续启用门禁约束。

## 验证边界

P2-02 覆盖关闭投影原子性、邀请/响应幂等、到期、低分分支、跟进状态、边界确认、匿名 DTO、跨机构拒绝、初始化/路由、Swagger/OpenAPI 和前端 lint/build。按阶段规则不做页面点触，阶段二集中验收由 P2-08 执行。

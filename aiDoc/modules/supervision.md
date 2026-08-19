# Supervision 模块边界

## 职责

`supervision` 为授权上级提供每日汇总、历史版本、复核队列、追加指导、直接介入、匿名服务评价和低分质量跟进能力。它读取 `careClient`、`carePath` 与 `caseWork` 已有事实形成投影，不修改答卷、规则命中、咨询互动或下级历史行动。

当前阶段只处理固定测试记录。每日任务会为已结束的前一自然日生成机构级测试快照，但正式报表、真实通知、医疗内容和面向用户的 AI 均保持关闭。

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

历史汇总是追加式不可变版本，以 `organizationId + businessDate + version` 唯一。每条记录保存指标口径版本、生成方式、生成/截止时间、来源摘要、前一版本、修正原因、指标差异和重点事项 JSON 快照；相同业务日期的修正只能创建更高版本。

`P2-04-v2` 的十二项指标为：

- `servedClients`：业务日内存在到期任务的去重服务人数；
- `dueTasks`：业务日内应执行任务数；
- `submittedTasks`：统计截止时点前提交的任务数；
- `overdueTasks`：统计截止时点前已到期且未提交或晚于到期时间提交的任务数；
- `deliveryIssues`：截止时点仍存在的送达异常待办数；
- `openAttentionCases`：截止时点未关闭的关注事项数；
- `resolvedAttentionCases`：业务日内已解决事项数；
- `consultationsOpened`：业务日内新增咨询数；
- `consultationsClosed`：业务日内关闭咨询数；
- `openConsultations`：截止时点仍未关闭的咨询数；
- `openTodos`：截止时点仍活动的统一待办数；
- `reviewRequired`：截止时点等待上级复核的事项数。

时区固定为 `Asia/Shanghai`。今日列表首项为请求时计算的 `REALTIME_PREVIEW`，`id=0` 且没有版本号；它不会写入历史表。历史项为 `VERSIONED_SNAPSHOT`，只能通过详情接口读取其冻结重点事项。

今日实时预览的截止点覆盖当前业务毫秒，因而与固定时钟恰好相同的已持久化提交、解决或关闭事实也会被纳入。历史快照仍使用自然日的半开区间，不改变历史口径。

生成方式为 `SCHEDULED|CORRECTION|SYSTEM_RECOMPUTE|LEGACY`。每日任务只创建缺失的 `P2-04-v2` 前一日快照；上级只能对已结束日期的当前最新版本发起修正。修正命令重新读取原始记录，只有来源摘要发生变化才追加新版本，并保存指标差异和重点事项是否变化；旧版本不更新。相同幂等键与相同请求重放原结果，不同请求复用键、过期版本或无变化复算均拒绝。

运营看板返回今日实时投影、最近 1–31 日的最新历史版本和缺失/修正覆盖情况。当前只做机构级统计，明确返回 `formalReportingEnabled=false` 和待确认的归属策略；不得据此生成人员排名、团队绩效或跨机构结论。

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

GORM 自动迁移包含 `daily_summary_versions`、`supervisor_guidance` 以及五张评价策略/邀请/响应/跟进/动作表。P2-04 只为现有日报版本表增加可回退字段和指标列，不覆盖已有 `P1-08-v1` 记录。固定测试开关启用时，初始化幂等登记 API、菜单、按钮、角色策略、评价策略版本，并仅在缺少目标记录时生成目标日期的 `P2-04-v2` 历史快照。

日报没有对外的初始生成接口。`CareDailySummary` 方法任务在固定测试门禁开启时登记，按 `CRON_TZ=Asia/Shanghai 10 0 * * *` 为所有活动测试机构补齐前一日快照；门禁关闭时不创建且会停用已有任务。任务使用系统身份，单机构失败不会阻止其他机构尝试，重复运行不重复建版本。

团队/人员转交归属、正式 KPI、跨机构分析和报表用途尚未获批，因此不进入当前数据模型或页面。历史修正只更正汇总投影，不能直接编辑指标，也不能替代原始事实修正流程。

## 验证边界

P2-04 覆盖十二项指标边界、今日/历史截止时点、跨机构隔离、自动任务系统身份与多机构幂等、修正版追加/差异/重放/冲突/无变化拒绝、初始化与路由、Swagger/OpenAPI 和前端 lint/build。按阶段规则不做页面点触，阶段二集中验收由 P2-08 执行。

# CarePath 模块

## 职责

`carepath` 负责版本化路径、计划模板、路径加入、计划实例、D1–D5 任务实例、任务开放协调、计划状态审计和事务型 outbox。P1-04 只启用一份生产关闭的 `TEST_ONLY` 合成 OSA 定义，用于验证计划预览、幂等启动和 `KEEP_WINDOWS` 暂停/恢复。

本模块不负责问卷题目与规则内容、康养用户会话与提交、关注事项、通知投递、正式医疗内容或面向用户的 AI。

## 后端入口

- Model：`server/model/carepath/`
- Service：`server/service/carepath/`
- API：`server/api/v1/carepath/`
- Router：`server/router/carepath/`
- 迁移与种子：`server/initialize/gorm_biz.go`、`server/initialize/care_path_seed.go`
- 共享 outbox：`server/internal/platform/outbox/`
- 责任范围决策：`server/internal/accesspolicy/careclient.go`
- 内容只读决策：`server/internal/accesspolicy/content.go`

后端保持 `Router -> API -> Service -> Model`。API 只处理 HTTP 绑定和统一响应；Service 只接收 `context.Context`，并把它传给所有 GORM 操作。运行表的部门和操作人归属由 DataScope 过滤、盖章；系统协调产生的事件必须显式继承计划的 `dept_id`。

## 前端入口

- API：`web/src/api/sleep-care/care-path.js`
- 计划模板页：`web/src/view/sleep-care/plans/index.vue`
- 任务页：`web/src/view/sleep-care/tasks/index.vue`
- 用户计划时间线：`web/src/view/sleep-care/clients/index.vue`
- 动态菜单组件名：`CarePlans`、`CareTasks`

计划模板和任务页只读。责任管家/医护可从用户详情选择明确的合成 `anchorAt`，先生成预览再确认启动，并可填写原因暂停或恢复；上级医师只读。

## 数据模型

全局不可变定义表不带行级归属列：

- `care_path_definition_versions`
- `care_plan_template_versions`
- `care_plan_task_definitions`
- `care_plan_task_dependencies`

运行表带标准 DataScope 归属列：

- `care_path_enrollments`
- `care_plan_previews`
- `care_plan_instances`
- `care_task_instances`
- `care_path_events`
- `care_path_command_receipts`
- `outbox_events`（与 questionnaire 共享）

定义哈希覆盖路径和计划的不可变业务内容。计划实例复制绝对时间窗、问卷版本、规则版本列表、复核角色、逾期策略和通知策略，历史运行事实不回读“当前模板”来改变语义。

## 服务接口

员工读取：

- `ListPlanVersions(ctx, search)` / `GetPlanVersion(ctx, id)`
- `ListClientPlans(ctx, careClientID)`
- `ListTasks(ctx, search)` / `GetTask(ctx, id)`

责任人命令：

- `PreviewPlan(ctx, careClientID, idempotencyKey, request)`
- `StartPlan(ctx, careClientID, idempotencyKey, request)`
- `PausePlan(ctx, planID, idempotencyKey, request)`
- `ResumePlan(ctx, planID, idempotencyKey, request)`

内部协调缝：

- `ReconcilePlanTasks(ctx, planID)`

所有命令把状态、审计事件、outbox 和幂等回执放在同一事务。可更新聚合使用整数 `version` 和 `expectedVersion`；重复键只回放同一请求的首次结果。

## P1-04 固定门禁

- 只接受 `OSA@1.0.0-synthetic` 与 `SYN-OSA-D1-D5@1.0.0-synthetic`。
- 路径和模板必须已发布、工程复核、`TEST_ONLY`、`synthetic=true`、`productionEnabled=false`，且合成夹具开关开启。
- 必须严格包含 D1–D5 五项 `CARE_CLIENT` 任务、零依赖、固定 24 小时日序和 11 小时窗口。
- D1 只允许问卷版本 `9401`、规则版本 `[9501]` 和医护复核；D2–D5 不绑定问卷、规则或复核。
- 问卷、规则、路径和计划定义哈希均在引用时验证；哈希不一致 fail closed。
- 通知策略固定 `DISABLED`，逾期策略固定 `DENY`，暂停策略固定 `KEEP_WINDOWS`。

## 状态与调度

- 路径加入/计划在 P1-04 只开放 `ACTIVE <-> PAUSED`。
- 任务执行、时效、复核是三个独立状态轴；`DENY` 下到达 `dueAt` 即推导 `EXPIRED`，不覆盖执行状态。
- 暂停期间不开放新任务；恢复后按原窗口补开已到 `openAt` 的任务，不平移时间窗。
- `TaskOpened`、`CarePlanStarted`、`CarePlanPaused`、`CarePlanResumed` 同时写审计事件和 outbox。
- `Clock` 可注入；测试使用固定时钟，禁止用运行时当前日期猜测 D1 锚点。

## 权限

- 当前有效责任管家/医护：读取责任用户并执行预览、启动、暂停、恢复。
- 上级医师：在组织树 DataScope 内只读。
- 内容管理员：只读方案定义版本；客户计划、任务和全部运行态接口继续拒绝。
- 同团队未负责人、跨机构角色、普通管理员 `888`、缺身份或未映射角色：fail closed。
- 菜单、按钮和 Casbin 只是第一层；Service 必须再次校验稳定领域角色、DataScope、当前责任关系、合成用户和合成运行记录。

## 必须保持的边界

- 不加入正式 D1–D5、临床阈值、诊断或治疗建议。
- 不开放填写、提交、补填、跳过、取消、重排、终止或模板升级动作。
- 不接入真实短信、外部通知或面向康养用户的 AI。
- 不让前端字段决定机构、部门或责任范围。
- 不在没有明确授权时把 `CARE_STEWARD` / `CLINICIAN` 任务定义变成运行夹具。
- 不在 P1-04 自动启动 P1-05 或后续闭环。

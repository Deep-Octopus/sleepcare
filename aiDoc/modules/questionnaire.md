# Questionnaire 模块

## 职责

`questionnaire` 是问卷定义、规则定义、答卷事实和规则命中的领域模块。P1-03 将定义读取与答卷写入收敛在同一深模块内，但只向后台 HTTP 暴露定义预览；提交能力留给后续任务模块和受限会话模块通过 Go 服务接口调用。

本模块不创建计划任务、移动端会话、关注事项、待办、通知或 AI 能力。

## 后端入口

- Model：`server/model/questionnaire/`
- Service：`server/service/questionnaire/`
- API：`server/api/v1/questionnaire/`
- Router：`server/router/questionnaire/`
- 权限决策：`server/internal/accesspolicy/questionnaire.go`
- 迁移与种子：`server/initialize/gorm_biz.go`、`server/initialize/questionnaire_seed.go`

后端继续遵循 `Router -> API -> Service -> Model`。Service 接收 `context.Context`，所有受控业务表查询和写入都使用 `WithContext(ctx)`，行级归属由 DataScope 回调处理。

## 前端入口

- API：`web/src/api/sleep-care/questionnaires.js`
- 页面：`web/src/view/sleep-care/questionnaires/index.vue`
- 动态菜单组件名：`CareQuestionnaires`

页面只读展示生命周期、使用范围、复核、生产门禁、题目、选项、校验和规则快照，不提供发布、禁用、填写或更正入口。

## 数据模型

定义表不带 `dept_id`，因为问卷和规则版本是全局内容配置：

- `questionnaire_versions`
- `questionnaire_questions`
- `questionnaire_options`
- `questionnaire_rule_versions`

运行事实表带 `dept_id` 和 `created_by`，由 DataScope 自动过滤和盖章：

- `questionnaire_submissions`
- `questionnaire_answer_revisions`
- `questionnaire_rule_hits`
- `questionnaire_command_receipts`
- `outbox_events`

`QuestionnaireSubmission.task_id` 在 P1-03 是外部冻结任务标识，不创建 P1-04 的任务表或外键。答卷唯一约束为一个任务一份提交；更正通过新增 `QuestionnaireAnswerRevision` 保留历史。

## 服务接口

公开读取：

- `ListVersions(ctx, search)`
- `GetVersion(ctx, id)`

供后续模块组合的写侧接口：

- `RecordSubmission(ctx, frozenBinding, command)`
- `AppendAnswerRevision(ctx, command)`

`FrozenTaskBinding` 必须显式携带问卷版本、规则版本列表、康养用户、任务和归属部门。评估只读取该列表，不扫描后来发布的规则，因此不会追溯改变旧任务。

## 生命周期和门禁

内容生命周期只允许：

```text
DRAFT -> IN_REVIEW -> APPROVED -> PUBLISHED -> DISABLED
```

- `PUBLISHED` 定义用定义哈希校验，服务不暴露原地更新接口。
- `TEST_ONLY` 必须同时满足合成标记、生产未启用、工程复核和本地合成夹具开关。
- `FORMAL` 必须满足非合成、生产启用和正式复核；P1-03 不提供正式内容、正式写入口或正式夹具。
- 未发布或已禁用的绑定规则被跳过，答卷仍成功并返回 `ruleExecutionDisabled=true`，对应内部 `41404` 语义。

## 答卷、修订、命中与事件

- 支持 `SINGLE_CHOICE`、`MULTIPLE_CHOICE`、`TEXT`、`NUMBER`、`DATE`、`BOOLEAN` 六类基本题型。
- 答案按绑定版本校验并以规范 JSON 保存。
- 工作人员代填必须记录来源原因和确认方式。
- 提交和修订使用操作人、操作、幂等键和请求哈希防止重复或错用。
- 规则命中保存条件、级别、原因和接收角色快照；同一提交、规则和去重键最多一条。
- 修订不覆盖旧答案，也不自动撤销、替换或重新生成旧 `RuleHit`。
- 首次提交与规则命中在同一数据库事务写入 `TaskAnswerSubmitted`、`RuleHitRecorded` outbox 事件；P1-03 不实现消费者。

## 权限

- 合成一线医护、合成上级医师：允许菜单、按钮、Casbin 和服务端内容预览。
- 合成健康管家、普通管理员 `888`、缺身份或未映射角色：默认拒绝。
- 三类合成业务角色只共享进入/退出后台所需的取菜单、取用户信息和 JWT 退出最小壳层权限；这些权限不读取问卷数据。
- 前端菜单和按钮不是安全边界；`internal/accesspolicy` 继续 fail closed。

## 合成种子

合成夹具开关启用时创建固定问卷版本 `9401`、固定规则版本 `9501` 及其题目、选项。种子幂等且严格：固定 ID 被占用或实际定义哈希不一致时返回错误，不覆盖已有内容。夹具关闭时只保留 API 和菜单元数据，不创建内容行。

## 必须保持的边界

- 不把 P1-03 的 `task_id` 扩展成计划或任务聚合。
- 不注册公开或移动端提交路由。
- 不根据数据库中“当前所有规则”重算历史答卷。
- 不把 `RuleHit` 当作已确认的关注事项或诊断结论。
- 不加入真实个人数据、真实医疗内容、真实短信或面向用户的 AI。

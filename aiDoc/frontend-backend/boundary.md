# 前后端边界说明

## 归属边界

- 后端负责路由、参数校验、业务逻辑和响应结构
- 前端负责页面流程、交互体验、本地状态和展示层
- 共同行为通过明确的 API 契约协作，不通过隐式约定耦合

## 契约规则

- 保持统一响应结构：`{ code, data, msg }`
- 保持统一分页结构：`{ page, pageSize, total, list }`
- 字段名不要随意漂移
- 前后端字段类型必须保持一致
- 后端必须提供完整而准确的 Swagger 接口说明
- 前端接口调用应以实际 Swagger 与后端实现为准

## 变更规则

- 涉及破坏性接口调整时，要先写清楚变更范围
- Swagger 或其他接口说明必须与真实实现一致
- 前端接口封装应继续放在 `web/src/api/` 或 `web/src/plugin/<name>/api/`
- 可复用逻辑优先复用 `web/src/utils/` 现有能力

## 完成前检查

跨前后端改动结束前，至少确认以下几点：

1. 后端响应结构仍然满足前端预期
2. 前端仍在使用正确的字段名和数据类型
3. 若契约发生了长期变化，对应说明已经补到 `aiDoc/`

## P1-03 问卷版本只读契约

- `GET /care/questionnaire-versions` 返回 `{page,pageSize,total,list}`，列表项使用 `lifecycleStatus`、`reviewRecord`、`questionCount` 和 `ruleCount`。
- `GET /care/questionnaire-versions/{id}` 返回同一版本摘要，并追加 `questions`、`rules`、`definitionSchemaVersion` 和 `replacesVersionId`。
- 题目顺序字段统一为 `order`；校验定义使用 `validationSchemaVersion` 和 JSON 对象 `validation`。
- 规则使用 `lifecycleStatus`、`reviewRecord`、`condition`、`attentionLevel`、`reasonSnapshot`、`recipients` 和 `dedupKeyTemplate`。
- 两个接口只允许医护和上级医师角色；前端 `CareQuestionnaires` 页面只读，不提供内容发布或答卷提交。
- 答卷写侧是后端 Go 服务边界，P1-03 不注册 HTTP 接口；P1-05 接入前不得从前端绕过该边界。

## P1-04 OSA 路径、计划与任务契约

- `GET /care/plan-versions` 返回 `{page,pageSize,total,list}`；列表项包含路径/模板版本、发布门禁、锚点定义、`DENY` 逾期策略、`KEEP_WINDOWS` 暂停策略和任务数。
- `GET /care/plan-versions/{id}` 追加严格有序的 D1–D5 定义；相对时间统一用 `openOffsetSeconds` / `dueOffsetSeconds`，不由前端推算漂移日期。
- `POST /care/clients/{id}/plan-previews` 必须携带 `Idempotency-Key`，请求传模板版本与带时区的明确合成 `anchorAt`；响应返回 opaque `previewId`、有效期和服务端计算的绝对任务时间窗。
- `POST /care/clients/{id}/plan-instances` 使用同一个预览 ID 和 `expectedClientVersion` 幂等启动；重复消费同一预览返回原计划，不创建第二套任务。
- `GET /care/clients/{id}/plan-instances` 返回该用户可见的计划、D1–D5 任务和审计时间线，供详情页刷新；这是 P1-04 为可恢复页面流程补充的员工只读接口。
- `POST /care/plan-instances/{id}/pause|resume` 都要求 `Idempotency-Key`、`expectedVersion` 和非空原因；只支持 `KEEP_WINDOWS`，前端不得显示或暗示时间窗被平移。
- `GET /care/tasks` 使用统一分页，可按用户、计划、日序、执行状态和派生时效状态筛选；`GET /care/tasks/{id}` 只返回冻结绑定与状态时间线，不返回答卷或医疗内容。
- 任务始终分别返回 `executionStatus`、由后端时钟派生的 `timingStatus`、`reviewStatus`；前端不得把提交、复核和时效合并成单一状态。
- D1 冻结 `questionnaireVersionId=9401` 与规则版本 `[9501]`；D2–D5 的 `questionnaireVersionId=null`、规则列表为空。所有 P1-04 运行夹具均为 `CARE_CLIENT`，通知策略固定 `DISABLED`。
- 任务开放事件名严格使用 P1-01 契约的 `TaskOpened`；计划状态事件与任务开放事件都在同一业务事务中写入审计时间线和共享 `outbox_events`。
- 管家、医护、上级可读计划版本和责任范围任务；只有当前责任管家/医护可预览、启动、暂停、恢复。上级在 P1-04 为只读，普通管理员与缺身份请求 fail-closed。

## P1-05 移动端受限访问与提交契约

- `POST /care/client-access/redeem` 只接收一次性 `grant`，成功通过受限 Path 的 HttpOnly Cookie 建立会话；响应体不返回 session token。
- `GET /care/client/tasks`、`GET /care/client/tasks/{taskId}` 只返回当前会话客户与任务白名单的交集，不接受 `clientId`、`organizationId` 或计划 ID 扩权。
- 任务列表/详情使用客户端专用 DTO，不返回客户身份、组织、计划内部标识、规则、审核记录或内部环境标志。
- `GET /care/client/tasks/{taskId}/questionnaire` 仅在确认并开始填写后返回冻结题目和可空 `draft`；不返回规则条件。草稿响应中的 `version` 是草稿版本，`taskVersion` 是最终提交使用的任务版本。
- `POST .../interactions` 按 `OPENED -> CONSENTED -> STARTED` 记录独立事实；每次写操作都要求 `Idempotency-Key`、任务乐观锁版本和精确同源 `Origin`。
- `PUT .../draft` 的 `answers` 是以问题编码为键的 JSON 对象，初始 `expectedVersion=0`；断网时前端保留本地进度，恢复联网后用新幂等键同步。
- `POST .../submit` 仅允许 `source=CLIENT_SELF`，在同一事务内创建答卷、规则命中及其关注事项并把任务转为 `SUBMITTED`；成功文案固定为“已提交，等待处理”。没有命中时 `attentionCaseIds` 为空，有命中时返回去重事项 ID；该字段不表示事项已处理。
- 客户端 API 通过 `authContext: 'client'` 复用请求工具，不发送员工头；客户端 401 只回到访问失效页，不清理或重定向员工登录状态。

## P1-06 提交触发关注事项契约

- 员工列表与详情使用 `GET /care/attention-cases` 和 `GET /care/attention-cases/{id}`；列表使用统一分页，只包含最小事项摘要，详情追加规则命中快照与行动时间线，不返回原始答案。
- 所有动作要求 `Idempotency-Key` 和 `expectedVersion`。确认、处理记录、升级、关闭和重开分别使用独立子路径；处理记录额外要求 `actionType=CONTACT|HANDLING`。
- 管家只能写 `CONTACT`，医护只能写 `HANDLING`；升级目标必须是当前有效责任医护。主管重开会创建新待办并清空上一轮可关闭字段，但不会删除历史行动。
- 没有处理结果返回 `41502`，没有关闭理由返回 `41503`；不存在、跨部门、无责任关系和缺身份均按失败关闭处理。
- 本任务只注册后端 API 与 Casbin path，不创建菜单、按钮或员工页面；这些入口由 P1-07 集中实现。

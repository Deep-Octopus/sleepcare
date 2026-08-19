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
- 两个接口允许医护、上级医师和内容管理员；前端 `CareQuestionnaires` 页面只读，不提供内容发布或答卷提交。
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
- 管家、医护、上级和内容管理员可读计划版本；内容管理员不能读取客户计划或任务。只有当前责任管家/医护可预览、启动、暂停、恢复；上级运行态只读，普通管理员与缺身份请求 fail-closed。

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
- 管家只能写 `CONTACT`，医护只能写 `HANDLING`。管家提交 `nextStatus=WAITING_COLLABORATION` 时，后端在同一事务中自动转交当前有效责任医护并返回两次状态变化后的最终版本；缺少医护时整笔提交回滚。手工升级目标仍必须是当前有效责任医护。主管重开会创建新待办并清空上一轮可关闭字段，但不会删除历史行动。
- 没有处理结果返回 `41502`，没有关闭理由返回 `41503`；不存在、跨部门、无责任关系和缺身份均按失败关闭处理。
- 本任务只注册后端 API 与 Casbin path，不创建菜单、按钮或员工页面；这些入口由 P1-07 集中实现。

## P1-07 管家与医护工作台契约

- `GET /care/workbench` 返回 `dueToday`、`waitingClient`、`deliveryIssues`、`attentionCases`、`assignedToMe`、`reviewRequired` 六个整数计数；响应不是持久化日报，前端必须按实时数据展示。
- 工作台计数只覆盖当前角色的部门范围与有效责任关系。前端的筛选、菜单和按钮不能扩大后端授权。
- 工作台、执行管理、计划任务和关注事项使用动态菜单元数据；工作台和关注事项页面使用独立按钮 key，操作可见性与后端动作权限保持一致。
- 关注事项列表只展示最小摘要，详情只追加规则命中快照与行动时间线，不展示原始答卷。
- 管家使用 `CONTACT` 记录联系，并可在同一次提交中选择自动转交责任医护；医护使用 `HANDLING` 记录处置。医护请求上级复核时写入 `WAITING_SUPERVISOR`，活动待办继续保持开放。
- P1-07 不新增指导、日报或通知写接口；相关指标只读取现有任务、事项和待办投影。

## P1-08 上级督导与每日汇总契约

- `GET /care/daily-summaries` 使用统一分页；今天的第一页首项可返回 `id=0`、无 `version` 的 `REALTIME_PREVIEW`，其余项目为有稳定 ID 和版本号的 `VERSIONED_SNAPSHOT`。
- `businessDate` 使用 `YYYY-MM-DD` 和 `Asia/Shanghai` 自然日。实时项由当前授权记录计算且不持久化；历史项只追加新版本，不更新旧版本。
- `GET /care/daily-summaries/{id}` 只接受历史版本 ID，返回冻结指标和 `focusCases`；前端不得把实时预览冒充历史日报，也不得自行拼接历史重点事项。
- `GET /care/reviews` 返回 `PENDING|GUIDED|INTERVENED|COMPLETED` 复核投影、原请求时间与请求人。详情继续通过已授权的关注事项详情接口读取，不返回原始答卷。
- `POST /care/reviews/{id}/guidance` 接收 `expectedVersion`、`guidance`、`responsibleAssigneeId` 和未来 `dueAt`；指导和讨论安排均追加 `GUIDANCE` 事实，事项继续处于 `WAITING_SUPERVISOR`。
- `POST /care/reviews/{id}/intervene` 接收 `expectedVersion`、`result`、`responsibleAssigneeId` 和未来 `dueAt`；成功后事项进入 `HANDLING`。
- 两个写接口都要求 `Idempotency-Key`。前端在一次对话中复用同一键，服务端负责相同请求重放、不同请求冲突和事项版本冲突。
- 五个接口只授权上级角色，并叠加业务身份、DataScope、机构范围和固定测试记录门禁。页面菜单或按钮不可替代后端权限。
- 当前没有日报生成 HTTP 接口和定时任务；初始化仅幂等提供一条固定历史版本。通知异常写侧仍由 P1-09 负责。

## P1-09 通知尝试、补发与人工联系契约

- `GET /care/deliveries` 使用统一分页，可按 attempt 状态筛选；每项返回 `version`、补发来源、请求/提交/受理/送达/终结时间和按时间排序的 `events`。
- `ACCEPTED` 的 `deliveredAt` 必须为空，前端必须显示“尚未确认送达”；客户端打开、开始和提交事件不得出现在通知回执时间线中。
- `POST /care/deliveries/{id}/resend` 只接受 `FAILED|UNKNOWN`，要求 `Idempotency-Key`、旧 attempt 的 `expectedVersion` 和非空原因；响应中的 `resourceId` 是新 attempt ID。
- `POST /care/tasks/{id}/contact-records` 要求任务 `expectedVersion`、`PHONE|OFFLINE|OTHER`、实际联系结果和带时区发生时间；成功只增加任务版本并追加时间线，不改变执行状态。
- 通知列表允许管家、医护和上级按责任/管理范围读取；补发仅当前责任管家，联系记录仅当前责任管家或医护。前端按钮不可替代后端责任关系校验。
- 阶段一 adapter 不访问网络，不接收手机号或通知正文。失败/未知形成统一送达异常待办；同一逻辑请求重试不重复建活动待办，也不改写 D1–D5。

## P1-10 完整权限、菜单与数据隔离契约

- P1-10 不新增 HTTP 接口；它在所有阶段一模块登记完成后，以显式白名单收敛四类员工角色的菜单、按钮和 Casbin 策略。
- 员工菜单固定为工作台、康养用户、执行管理、内容管理和督导中心。内容管理员只收到内容管理下的问卷内容与服务计划；不收到客户、执行、通知或督导路由。
- `CareClientDetail`、`CareTaskDetail`、`CareAttentionCaseDetail` 和 `CareReviewDetail` 是隐藏动态路由，分别用 `activeName` 指向所属列表；四个组件名稳定且唯一。
- 默认首页必须是角色已授权的可见叶子：管家/医护为 `CareWorkbench`，上级为 `CareDailySummaries`，内容管理员为 `CareQuestionnaires`。
- 登录成功后前端必须在当前账号的动态菜单树中验证默认首页，忽略登录前旧账号遗留的 `redirect`，清空会话页签并重新加载动态路由；默认首页缺失时停留并提示管理员配置，不跳转旧页面。
- 问卷版本只读允许医护、上级和内容管理员；方案版本只读允许管家、医护、上级和内容管理员。内容管理员仍被客户与所有运行态服务拒绝。
- 动态菜单与按钮只控制导航和操作可见性。手工输入路由后，后端仍依次要求员工身份、Casbin、DataScope、领域角色和当前责任关系。
- 固定角色的权限初始化会移除陈旧授权并幂等重建；系统管理角色 `888` 不因可管理 RBAC 而获得 `/care/**` 业务数据权限。

## P1-11 阶段一验收时钟与运行契约

- 日常 Compose 默认以显式空的 `GVA_CARE_FIXTURE_NOW` 使用系统时钟；只有显式提供 RFC3339 值时，`care.fixture-now` 才在本地固定记录开关下接管业务时钟，非法值必须阻止启动。
- 显式空环境值必须优先于持久化配置，以便不重置配置卷即可退出固定时钟模式。
- 客户受限会话的剩余寿命以业务时钟计算；Cookie 的 HTTP `Expires` 将该时长映射到操作系统时钟，避免固定时钟让有效会话立即过期。
- 今日实时汇总包含与当前业务毫秒相同的已持久化事实；历史快照仍按完整自然日截止点计算。
- P1-11 不新增生产 API 或数据库迁移；本地浏览器截图与 trace 必须保持在已忽略目录，不得带入会话或一次性访问凭据。

## P2-01 主动咨询契约

- 客户受限会话使用 `POST /care/client/consultations` 发起咨询，使用列表、详情和 `messages` 子路径查看本人公开互动并追加补充；请求不能携带 `clientId`、机构或责任人扩权。
- 客户 DTO 只返回主题、联系顺序、状态、时间、版本、解决结果/后续安排和公开互动；内部转交原因、责任目标和工作人员内部标识不返回。
- 员工端使用 `/care/consultations` 列表与详情；列表遵循统一分页，详情返回咨询摘要、初始问题、解决/关闭字段和完整互动时间线。
- `GET /care/consultations/{id}/assignee-options` 只返回该咨询当前有效管家/医护和同机构有效上级，供分配、转交、升级选择；前端不得手填人员 ID 或自行扩大候选范围。
- 所有客户与员工写操作都要求 `Idempotency-Key` 和 `expectedVersion`；创建请求没有 `expectedVersion`，由服务端创建版本 `1`，自动分配时返回版本 `2`。
- 当前责任人才能回复、转交、升级和解决；上级可分配待分配咨询、关闭已解决咨询和重开已关闭咨询。后端责任校验优先于前端按钮状态。
- 状态值为 `NEW`、`WAITING_ASSIGNMENT`、`ASSIGNED`、`HANDLING`、`WAITING_CLIENT`、`WAITING_COLLABORATION`、`RESOLVED`、`CLOSED`；前端只做文案映射，不自行推导状态变化。
- `ROUTINE|EXPEDITED` 只表示联系处理顺序，不是医疗紧急程度或响应承诺。移动端必须同时显示随时接收、人工回复按服务安排和正式急救/就医渠道提示。
- `P2-01` 不发起满意度、不接入电话/录音、真实短信或用户侧 AI；页面点触统一留到阶段二收口。

## P2-02 服务评价契约

- `caseWork` 在咨询关闭事务内调用 `ConsultationClosedProjector`；评价邀请按关闭事件唯一，投影失败会回滚关闭，补偿扫描重复执行不会重复创建。
- 邀请冻结 `policyCode`、`policyVersion`、匿名方式、有效期和低分阈值；客户端只使用服务端返回的状态、版本和时间，不自行推导到期或低分规则。
- 客户端通过 `/care/client/satisfaction-requests` 列表/详情和 `responses` 写路径访问本人评价；请求不接受用户、机构、咨询或服务责任人 ID。
- 客户端写操作要求 `Idempotency-Key` 和 `expectedVersion`。前端持久复用未确认提交的键，成功后清理；服务端负责同请求重放、不同内容冲突、重复响应和到期拒绝。
- 上级通过 `/care/satisfaction-responses` 与 `/care/satisfaction-follow-ups` 查看匿名评价和低分跟进；DTO 不返回用户、咨询、关闭事件或服务责任人关联。
- 质量跟进状态为 `OPEN|IN_REVIEW|RESOLVED`。接收/解决都要求幂等键和版本；解决额外要求 `usageBoundaryConfirmed=true` 并在同一事务完成活动待办。
- `CareSatisfaction` 菜单、三个按钮和五个员工 API 只授权上级，并叠加 DataScope、机构和质量责任校验。客户端路由继续使用受限会话和严格同源校验。
- 当前只提供站内邀请、匿名列表和个案质量跟进，不提供外部发送、电话评价、正式排名、人员结论、真实短信或用户侧 AI；页面点触留到 P2-08。

# Case Work 模块边界

## 职责

`casework` 把已发布规则产生的 `RuleHit` 转换为员工可处理的关注事项，并负责事项状态、追加式行动记录、统一待办、乐观锁、写操作幂等、outbox 事实和员工责任范围工作台聚合。

模块不生产问卷答案或规则结论，不修改原答卷，不发送短信，不提供面向用户的 AI，也不拥有上级指导记录、日报或通知通道。`supervision` 只能通过本模块的事务 seam 追加事项行动并更新活动待办，不能绕过事项状态机。

## 后端入口

- Model：`server/model/casework/`
- Service：`server/service/casework/`
- API：`server/api/v1/casework/`
- Router：`server/router/casework/`
- 前端工作台：`web/src/view/sleep-care/workbench/`
- 前端事项队列：`web/src/view/sleep-care/attention-cases/`
- 前端接口封装：`web/src/api/sleep-care/case-work.js`
- 迁移、API 元数据和受控补偿：`server/initialize/gorm_biz.go`、`server/initialize/case_work_seed.go`
- 责任范围策略：`server/internal/accesspolicy/careclient.go`
- 上级动作事务 seam：`server/service/casework/supervisor.go`

员工路由进入标准 PrivateGroup 中间件链；读取还要经过部门范围与有效责任关系的交集，动作接口再校验当前责任人和角色。

## 提交事务边界

客户端提交由 `clientaccess.SubmitTask` 持有最外层事务。`questionnaire.RecordSubmission` 追加答卷、修订、规则命中和提交事件；随后 `casework.OpenFromRuleHits` 为每个命中去重创建事项、活动待办和 `AttentionCaseOpened` outbox 事件。任一步失败，答卷、命中、事项、待办、任务状态和事件一起回滚。

没有命中时提交正常成功且返回空 `attentionCaseIds`。有命中但缺少有效责任管家时返回 `41201`，提交不产生半成品。

## 数据与状态不变量

- `AttentionCase` 以 `sourceType + sourceRuleHitId + dedupKey` 唯一，重复提交或补偿不会创建第二条事项。
- 每个事项最多一条 `activeSlot=ACTIVE` 的待办；升级会终结旧待办并创建目标责任人的新待办，关闭会完成活动待办，重开会新建待办。
- 责任管家以 `CONTACT + WAITING_COLLABORATION` 提交联系结果时，Service 先解析当前有效责任医护，再在同一幂等事务中追加联系与系统转交两条行动、替换活动待办并返回最终版本；缺少责任医护时整笔事务回滚。
- 升级目标必须是责任链中另一位当前有效责任医护；不得把事项升级给当前责任人，也不得因此把待上级复核事项退回协作状态。
- `CaseAction` 只追加，不覆盖。它保存操作者、当时角色、机构/团队、动作来源、结果、理由、前后状态和发生时间。
- 所有员工写动作要求 `Idempotency-Key` 和 `expectedVersion`；相同键同请求返回原结果，相同键不同请求拒绝，过期版本拒绝。
- `PENDING_ACK -> ACKNOWLEDGED -> HANDLING/WAITING_* -> RESOLVED -> CLOSED`；主管可以把 `CLOSED` 重开为 `HANDLING`，原行动保留，上一轮处理结果与关闭字段不作为新一轮关闭依据。
- 正式等级和 SLA 尚未启用；当前只处理已获准的 `TEST_ONLY` 记录。

## 角色边界

- 责任管家：确认并记录 `CONTACT`；需要医护继续处理时由系统自动转交当前有效责任医护。既有手工升级接口保留为受控兼容入口。
- 当前责任医护：确认、记录 `HANDLING`、升级、在结果完整并已解决后关闭。
- 授权主管：在部门管理范围内读取、关闭、重开，并由 `supervision` seam 对等待复核事项追加指导或介入。
- 普通管理员、缺少康养业务身份、跨部门人员和无有效责任关系人员均失败关闭。

## 员工接口

- `GET /care/workbench`
- `GET /care/attention-cases`
- `GET /care/attention-cases/{id}`
- `POST /care/attention-cases/{id}/acknowledge`
- `POST /care/attention-cases/{id}/handling-records`
- `POST /care/attention-cases/{id}/escalate`
- `POST /care/attention-cases/{id}/close`
- `POST /care/attention-cases/{id}/reopen`

列表只返回事项摘要，不返回原始答案。详情返回命中原因快照和行动时间线，不返回答卷内容。

## 工作台投影

`GET /care/workbench` 在请求时实时聚合六项计数：`dueToday`、`waitingClient`、`deliveryIssues`、`attentionCases`、`assignedToMe` 和 `reviewRequired`。日期边界使用 `Asia/Shanghai`；所有查询先收敛到当前部门范围与有效责任关系，再统计已启用的固定测试记录。

工作台不持久化快照，也不新增业务状态。通知异常只读取统一待办中的 `DELIVERY_ISSUE` 分类；其创建和重试由通知模块负责。医护通过已有处置接口写入 `WAITING_SUPERVISOR` 请求上级复核，活动待办继续开放，直至后续动作完成。

页面按钮只控制交互可见性，后端仍以 Casbin、DataScope、有效责任关系、当前责任人与动作角色共同授权。管家在一次联系提交中即可选择自动转交，医护可记录专业处置或请求复核，主管在本任务中只有读取、关闭和重开既有能力。

## 受控补偿

启动补偿只在固定测试数据开关启用时扫描既有 `RuleHit`，且只处理对应测试提交。补偿沿用同一唯一键和创建服务，幂等补齐事项、待办与 outbox；开关关闭时不执行，也不触碰正式记录。

## 验证边界

阶段内只执行当前改动直接相关的 casework、初始化、路由、契约、前端定向 lint 和生产构建。上级指导与日报已由 P1-08 通过事务 seam 接入，通知尝试与重试属于 P1-09；页面点触留到阶段集中验收。

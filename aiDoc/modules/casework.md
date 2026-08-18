# Case Work 模块边界

## 职责

`casework` 把已发布规则产生的 `RuleHit` 转换为员工可处理的关注事项，并负责事项状态、追加式行动记录、统一待办、乐观锁、写操作幂等和 outbox 事实。

模块不生产问卷答案或规则结论，不修改原答卷，不发送短信，不提供面向用户的 AI，也不包含员工工作台页面、上级指导、日报或通知通道。

## 后端入口

- Model：`server/model/casework/`
- Service：`server/service/casework/`
- API：`server/api/v1/casework/`
- Router：`server/router/casework/`
- 迁移、API 元数据和受控补偿：`server/initialize/gorm_biz.go`、`server/initialize/case_work_seed.go`
- 责任范围策略：`server/internal/accesspolicy/careclient.go`

员工路由进入标准 PrivateGroup 中间件链；读取还要经过部门范围与有效责任关系的交集，动作接口再校验当前责任人和角色。

## 提交事务边界

客户端提交由 `clientaccess.SubmitTask` 持有最外层事务。`questionnaire.RecordSubmission` 追加答卷、修订、规则命中和提交事件；随后 `casework.OpenFromRuleHits` 为每个命中去重创建事项、活动待办和 `AttentionCaseOpened` outbox 事件。任一步失败，答卷、命中、事项、待办、任务状态和事件一起回滚。

没有命中时提交正常成功且返回空 `attentionCaseIds`。有命中但缺少有效责任管家时返回 `41201`，提交不产生半成品。

## 数据与状态不变量

- `AttentionCase` 以 `sourceType + sourceRuleHitId + dedupKey` 唯一，重复提交或补偿不会创建第二条事项。
- 每个事项最多一条 `activeSlot=ACTIVE` 的待办；升级会终结旧待办并创建目标责任人的新待办，关闭会完成活动待办，重开会新建待办。
- `CaseAction` 只追加，不覆盖。它保存操作者、当时角色、机构/团队、动作来源、结果、理由、前后状态和发生时间。
- 所有员工写动作要求 `Idempotency-Key` 和 `expectedVersion`；相同键同请求返回原结果，相同键不同请求拒绝，过期版本拒绝。
- `PENDING_ACK -> ACKNOWLEDGED -> HANDLING/WAITING_* -> RESOLVED -> CLOSED`；主管可以把 `CLOSED` 重开为 `HANDLING`，原行动保留，上一轮处理结果与关闭字段不作为新一轮关闭依据。
- 正式等级和 SLA 尚未启用；当前只处理已获准的 `TEST_ONLY` 记录。

## 角色边界

- 责任管家：确认、记录 `CONTACT`、升级到当前有效责任医护。
- 当前责任医护：确认、记录 `HANDLING`、升级、在结果完整并已解决后关闭。
- 授权主管：在部门管理范围内读取、关闭已解决事项、重开已关闭事项。
- 普通管理员、缺少康养业务身份、跨部门人员和无有效责任关系人员均失败关闭。

## 员工接口

- `GET /care/attention-cases`
- `GET /care/attention-cases/{id}`
- `POST /care/attention-cases/{id}/acknowledge`
- `POST /care/attention-cases/{id}/handling-records`
- `POST /care/attention-cases/{id}/escalate`
- `POST /care/attention-cases/{id}/close`
- `POST /care/attention-cases/{id}/reopen`

列表只返回事项摘要，不返回原始答案。详情返回命中原因快照和行动时间线，不返回答卷内容。

## 受控补偿

启动补偿只在固定测试数据开关启用时扫描既有 `RuleHit`，且只处理对应测试提交。补偿沿用同一唯一键和创建服务，幂等补齐事项、待办与 outbox；开关关闭时不执行，也不触碰正式记录。

## 验证边界

阶段内只执行 casework、clientaccess、初始化、路由和契约的定向测试与编译检查。员工页面与按钮属于 P1-07，上级指导属于 P1-08；页面点触留到阶段集中验收。

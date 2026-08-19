# AI 辅助关闭态门禁

## 模块职责

P2-06 的 `aiassist` 模块只负责暴露工作人员 AI 影子能力的关闭状态。阶段二尚未收到启用选择，知识来源、人工审核、完整谱系、禁用场景、数据处理、评测和模型供应商也未完成审批，因此本模块不生成建议、不检索知识、不调用外部模型，也不提供发送能力。

## 入口与依赖

- 运行配置：`server/config/care.go` 的 `care.ai-shadow`
- 后端响应：`server/model/aiassist/response/readiness.go`
- 后端服务：`server/service/aiassist/readiness.go`
- 后端 API：`server/api/v1/aiassist/ai_assist.go`
- 后端路由：`server/router/aiassist/ai_assist.go`
- 初始化元数据：`server/initialize/ai_assist_seed.go`
- 依赖现有业务身份、JWT、改密保护、Casbin 和 DataScope，不依赖知识库、咨询正文或模型 SDK。

## 配置与状态

`care.ai-shadow.mode` 当前只接受 `DISABLED`。空值按 `DISABLED` 处理；`SHADOW`、`CONTRACT_TEST`、`PRODUCTION` 或其他值都会阻止启动。

准备状态固定返回：

- `selectionStatus=NOT_SELECTED`
- `usageScope=NOT_ENABLED`
- `staffShadowEnabled=false`
- `suggestionGenerationEnabled=false`
- `knowledgeRetrievalEnabled=false`
- `externalModelEnabled=false`
- `userFacingAiEnabled=false`
- `directSendEnabled=false`

知识来源、人工审核流程、完整谱系、禁用场景政策、数据处理评审、评测协议和模型供应商评审也全部返回未就绪，并通过稳定阻塞码逐项说明。

## 接口契约

| 方法 | 路径 | 契约 |
| --- | --- | --- |
| `GET` | `/care/ai-shadow-readiness` | 返回关闭模式、阶段二选择状态、能力开关、前置条件与阻塞码 |

本模块没有 propose、retrieve、review、adopt、reject 或 send 路由，也没有面向康养用户的接口。

## 权限与数据范围

- 管家、医护和上级可读取关闭态。
- 内容管理员、普通系统管理员和缺少有效业务身份的请求失败关闭。
- 路由仍经过 `JWT -> MustChangePwdGuard -> Casbin -> DataScope`。
- 接口不接受康养用户 ID、咨询 ID 或自由文本，不读取服务过程数据。

## 数据与迁移

本模块没有持久化实体，不创建 `KnowledgeArticleVersion`、`AISuggestion` 或 `FinalReply`，也不修改既有咨询和互动表。初始化只幂等登记一条只读 `SysApi` 元数据，并由既有权限收敛器登记三类工作人员的 Casbin 策略。

## 后续启用门禁

只有收到明确选择，并完成知识责任人、版本化来源、人工审核、谱系保存、禁用场景、数据处理、模型供应商和评测输入后，才重新冻结数据模型、状态机和接口。正式受限交付仍属于阶段三，任何低风险自动回复继续使用独立审批和停用机制。

## 必须保持的不变量

1. 配置不能把当前实现切换为活动影子模式。
2. 关闭态接口不能成为建议生成或发送入口。
3. 不接收或向外部系统传输用户上下文。
4. 不新增面向用户的 AI 能力或真实短信能力。
5. P2-06 的任务完成状态不能被解释为已经运行影子验证。

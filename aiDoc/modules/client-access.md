# Client Access 模块边界

## 职责

`clientaccess` 为康养用户提供与员工端完全分离的身份与访问边界：账号密码或一次性 grant 建立 HttpOnly 会话，并协调本人任务交互、问卷草稿与最终提交。账号会话覆盖本人的客户端服务，grant 会话只携带固定客户和任务范围。

模块不负责 grant 投递、真实短信、员工菜单、医疗内容生产或面向用户的 AI；最终提交只负责协调调用 `casework`，事项本身仍由该模块维护。

## 后端入口

- Model：`server/model/clientaccess/`
- Service：`server/service/clientaccess/`
- API：`server/api/v1/clientaccess/`
- Router：`server/router/clientaccess/`
- Session / Origin 中间件：`server/middleware/client_access.go`
- 迁移与固定账户：`server/initialize/gorm_biz.go`、`server/initialize/client_access_seed.go`

路由挂载在 `PublicGroup` 下。登录与兑换是公开入口，其中账号登录必须通过精确同源校验；其余接口必须先通过 `ClientSessionAuth`，客户端写请求再叠加精确同源校验。它们不得进入员工端 `JWTAuth -> MustChangePwdGuard -> CasbinHandler -> DataScope` 链，也不得接受员工 token 代替客户端会话。

## 前端入口

- 静态路由：`web/src/router/index.js` 的 `/client/**`
- API：`web/src/api/sleep-care/client-access.js`
- 独立布局与页面：`web/src/view/client/`

客户端请求仍复用 `web/src/utils/request.js`，通过 `authContext: 'client'` 禁止发送员工 `x-token` / `x-user-id`，并启用 Cookie 会话。客户端路由使用 `meta.client`，不加载员工动态菜单。

## 数据与安全不变量

- `CareClientAccount` 是独立安全主体，不是 `SysUser`。
- `CareClientCredential` 只保存规范化账号、bcrypt 哈希和失败锁定状态，不承载公开资料；密码不得进入响应、日志或源码。
- grant、session 和幂等键只保存 SHA-256 摘要；原文不得写入数据库、日志、文档或 Swagger 响应。
- grant 只允许从 `ISSUED` 原子转换到 `REDEEMED`，并由 `ClientSession.grantId` 唯一约束防止并发重复兑换。
- `ClientSession.authType=TASK_GRANT` 时必须存在唯一 `grantId` 并固化任务范围；`authType=ACCOUNT` 时 `grantId` 为空，可读取本人全部客户端任务。
- 无论会话来源，每次任务访问都必须同时满足有效 session、账户、客户归属、任务客户、执行角色和部门范围；grant 会话额外要求任务白名单。
- 账号连续失败五次后锁定十五分钟；失败次数更新和登录判定在同一行锁事务中完成，成功后清零。账号不存在与密码错误使用同一外部提示。
- 退出登录撤销当前服务端 session 并清理同一路径 Cookie。
- 客户端写请求必须匹配配置中的精确 `Origin`。Cookie 使用 `HttpOnly`、`SameSite=Lax`、受限 Path；正式配置默认 `Secure`。
- `OPENED`、`CONSENTED`、`STARTED`、`SUBMITTED` 是独立事实；`STARTED` 才把任务从 `OPEN` 转为 `IN_PROGRESS`。
- 草稿使用独立版本，最终提交使用任务版本。草稿提交后保留并标记 `consumedAt`，不覆盖最终答卷历史。
- 最终提交在一个外层事务中复用 questionnaire 与 casework 边界，原子写入答卷、修订、规则命中、事项、待办、outbox、任务状态和任务事件。
- 没有规则命中时 `attentionCaseIds` 为空；存在命中时返回已去重创建的事项 ID。缺少有效责任管家时整次提交回滚。

## 客户端 DTO

客户端身份 DTO 只返回显示名称、显示编码和会话到期时间；业务 DTO 只包含任务展示、状态、时间窗、冻结题目和可恢复草稿。不得返回组织、计划内部标识、规则条件、审核记录、定义哈希、密码状态或内部环境标志。

问卷接口只返回服务端冻结题目与可空草稿；已发布规则仍在提交服务内部运行，但不会下发给客户端。

## 配置与本地数据

`care.client-access` 配置会话时长、Cookie 名称/路径/Secure 和允许来源。固定本地数据创建一个客户端账户及其独立登录凭据，不创建、保存或打印可用 grant；凭据密码复用受保护的本地密码配置。服务测试在内存中创建短期 bearer 值。

固定业务时钟启用时，session 剩余寿命仍以业务时钟计算；API 将这个时长映射到当前 HTTP 时钟的 Cookie `Expires`，不直接把固定日期写入浏览器 Cookie。

## 验证边界

阶段内执行 clientaccess 服务测试、日志脱敏测试、相关初始化测试、Swagger 契约检查及前端 lint/build。页面点触留到阶段集中验收或用户另行明确要求。

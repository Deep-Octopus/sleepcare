# 试用准备与桌面演练契约

## 模块职责

P2-07 用版本化契约管理阶段二 UAT、角色培训、灰度门禁、暂停和回滚桌面演练。当前没有真实试用授权或运行保障输入，因此该模块只验证固定测试边界和失败关闭，不执行部署、切流、备份恢复或外部通知。

## 入口与依赖

- 机器契约：`docs/contracts/phase2-rehearsal.json`
- 说明与操作手册：`docs/阶段二P2-07UAT培训灰度与回滚演练.md`
- 契约测试：`server/config/phase_two_rehearsal_contract_test.go`
- 执行入口：`make phase2-rehearsal-check`
- 集中验收结果：`docs/contracts/phase2-acceptance.json`
- 集中验收说明：`docs/阶段二P2-08验收与试用复盘.md`
- 集中验收测试：`server/config/phase_two_acceptance_contract_test.go`
- 集中验收入口：`make phase2-acceptance-check`
- 依赖 P2-01～P2-06 已冻结的功能边界，不依赖数据库、HTTP、前端、Docker API 或外部服务。

## 契约边界

契约固定：

- `environment=LOCAL_TEST`
- `dataScope=FIXED_TEST_ONLY`
- `executionMode=TABLETOP`
- `realTrialEnabled=false`
- `promotionAllowed=false`

真实数据、正式通知、工作人员 AI 影子、面向用户的 AI 和全部外部调用均保持关闭。真实试用范围、正式内容、身份与同意、通知、SLA、容量、监控/事件责任人、备份 RPO/RTO、恢复演练和安全评审都是显式阻塞项；AI 影子选择为 `NOT_SELECTED`。

## UAT 与培训

契约登记 `P2-UAT-01`～`P2-UAT-10`，覆盖角色/机构边界、咨询、评价、通知契约、日报与修正、数据治理关闭态、AI 关闭态、暂停/恢复/重开、幂等重试和无外部副作用。每项均为 `DEFERRED_TO_P2_08`，P2-07 不提前制造页面或 Compose 验收结论。

培训材料覆盖客户受限会话、管家、医护、上级、内容管理员和普通管理员负向边界。当前状态固定为 `PREPARED_NOT_DELIVERED`，不记录真实姓名、联系方式、账号或凭据。

## 桌面演练状态

桌面状态只存在于版控契约中：

```text
HOLD -> DRY_RUN -> DRY_RUN_PAUSED -> DRY_RUN_ROLLED_BACK -> HOLD
```

在 `DRY_RUN` 中执行 `PROMOTE` 必须返回 `DENIED` 并停留原状态。演练禁止网络调用、部署、数据库写入、数据库恢复、卷重置和 Git 历史改写；回滚策略为 `CODE_ONLY_RETAIN_DATA`，只验证决策顺序，不执行回退命令。

## 接口、权限与迁移

- 不新增 HTTP、Swagger、菜单、按钮、Casbin 或 DataScope 规则。
- 不新增数据库表、字段、业务状态或迁移。
- Makefile 入口只运行严格 JSON 契约测试，不读取凭据，不调用 Docker，不修改运行环境。

## 必须保持的不变量

1. 未完成外部门禁前，真实试用和 promotion 始终关闭。
2. P2-07 的桌面演练不能被记录为真实发布、备份恢复或人员培训完成。
3. UAT 执行与浏览器证据只在 P2-08 集中验收时产生。
4. 任何回滚计划默认保留业务数据；删除、恢复或重置必须另行获得明确授权。
5. 契约不得包含真实身份、联系方式、凭据、医疗内容或外部服务调用参数。

## P2-08 集中验收边界

P2-08 复用十条 UAT 用例执行阶段级自动化、Swagger、前端生产构建、Compose/迁移幂等、运行态保护和必要浏览器检查。机器结果把工程结论与真实试用结论分开：浏览器证据缺失时必须保持 `BLOCKED_PENDING_BROWSER_AUTH`；浏览器通过后只能形成 `PASSED_WITH_KNOWN_REPOSITORY_DEBT`，不能自动解除真实试用或 promotion 门禁。

集中验收不新增数据库、HTTP、权限或业务状态，也不执行部署、切流、恢复、卷重置、历史改写或外部调用。全仓扫描中的非阶段二失败按独立技术债记录，不通过修改无关模块来制造全绿结论。

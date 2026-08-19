# 模块说明索引

## 推荐写法

每个重要模块或插件，尽量用一份文档回答下面几个问题：

1. 这个模块是做什么的
2. 它的后端入口在哪里
3. 它的前端入口在哪里
4. 它依赖哪些数据或其他模块
5. 它对外暴露什么契约
6. 它有哪些必须记住的限制

## 当前建议的模块分组

- `system-core`: 项目核心能力，主要分布在 `server/` 与 `web/src/`
- `plugins`: 插件化能力，分布在 `server/plugin/` 与 `web/src/plugin/`
- `compose`: 本地运行编排，位于根目录 `compose.yaml` 与前后端 Dockerfile

## 已存在的约束文档

- `backend-layer-rules.md`: 后端分层、模型、Service、API、Router、初始化入口
- `plugin-development.md`: 前后端插件结构、插件入口与开发流程
- `questionnaire.md`: 问卷/规则不可变版本、答卷修订、规则命中、outbox、权限和合成夹具边界
- `carepath.md`: 路径/计划/任务版本、D1–D5 调度、幂等命令、责任权限、共享 outbox 和合成门禁
- `client-access.md`: 一次性 grant、客户端会话、任务白名单、草稿/提交事务、独立移动路由和安全边界
- `casework.md`: 规则命中到关注事项、员工工作台、追加行动、统一待办、状态机、责任权限、幂等与受控补偿
- `supervision.md`: 十二项实时汇总、定时历史版本、追加修正、运营覆盖、复核投影、上级指导/介入与管理范围
- `notification.md`: 通知请求/尝试、供应商契约、请求签名、回调验签、重试/限流/费用边界、异常待办与任务联系记录
- `data-governance.md`: 数据治理准备状态、正式能力关闭门禁、追加式生命周期请求、权限与迁移边界
- `ai-assist.md`: 工作人员 AI 影子能力的关闭态、只读门禁、前置条件和后续启用边界
- `trial-rehearsal.md`: 阶段二 UAT/培训准备、灰度门禁和无副作用暂停/回滚桌面演练
- `phase-one-access-control.md`: 阶段一四角色菜单、按钮、Casbin、DataScope、责任关系、内容只读边界与隐藏详情路由收敛

## 命名建议

新增模块文档时，优先使用这类文件名：

- `system-core.md`
- `plugin-<name>.md`
- `deploy-runtime.md`

模块说明要聚焦职责和边界，不要堆砌实现细节。

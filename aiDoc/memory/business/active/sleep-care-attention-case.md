# 睡眠康养：关注事项闭环

## 基本信息

- 提出日期：2026-08-18
- 当前状态：`active`
- 确认状态：工程实施基线已确认；正式业务、医疗与真实能力按 `docs/需求文档 r2.md` 第 16.2 节门禁启用
- 需求类型：核心业务流程
- 优先级：P0
- 需求文件：`aiDoc/memory/business/active/sleep-care-attention-case.md`

## 用户原始意图摘要

答卷命中已批准规则后，形成可分配、确认、处置、升级、关闭和重开的关注事项。

## 影响范围

- 后端：`caseWork`、事项状态、行动记录和 SLA
- 前端：关注事项队列、详情、处置和督导下钻
- 文档：关注等级和 SLA 附件
- 插件 / 模块：普通 package

## 涉及对象

- 模块：caseWork
- 接口：acknowledge、record action、transfer、escalate、close、reopen
- 页面：关注事项列表/详情、工作台待办
- 配置：等级、责任人、确认/升级/解决时限

## 工程实施基线（真实能力受门禁约束）

- “命中规则”表示需要关注，不等同于诊断。
- 未记录处理结果和关闭理由不得关闭。
- 转交不重置 SLA，关闭后可授权重开且不覆盖历史。
- 列表只显示最小必要摘要，敏感答案在授权详情查看。

## 当前进展

- P1-01 已由项目负责人批准：冻结 AttentionCase 状态机、追加式 CaseAction、关闭门禁、错误码和场景 A 的责任链。

## 后续待办

- P1-06 实现幂等规则命中、事项去重、确认/处置/升级/关闭和重开测试。
- 正式等级、SLA 和升级链未批准前只允许 `SYNTHETIC_ATTENTION` 测试等级。

## 相关需求

- [问卷与规则版本](sleep-care-questionnaire-versioning.md)
- [每日汇总与督导](sleep-care-supervision.md)

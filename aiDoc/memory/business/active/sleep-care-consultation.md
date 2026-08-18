# 睡眠康养：主动咨询服务

## 基本信息

- 提出日期：2026-08-18
- 当前状态：`active`
- 确认状态：工程实施基线已确认；正式业务、医疗与真实能力按 `docs/需求文档 r2.md` 第 16.2 节门禁启用
- 需求类型：阶段二业务流程
- 优先级：P1
- 需求文件：`aiDoc/memory/business/active/sleep-care-consultation.md`

## 用户原始意图摘要

康养用户可以主动咨询，服务团队完成接收、分配、转交、升级、解决和关闭。

## 影响范围

- 后端：`caseWork` 咨询子域、值班和 SLA
- 前端：移动咨询、员工队列、互动时间线
- 文档：服务时段、急症提示和电话授权
- 插件 / 模块：普通 package；呼叫平台作为后续 adapter

## 涉及对象

- 模块：caseWork/consultation
- 接口：create、assign、reply、transfer、escalate、resolve、close、reopen
- 页面：联系服务、咨询队列、咨询详情
- 配置：服务时段、SLA、值班和升级联系人

## 工程实施基线（真实能力受门禁约束）

- 24 小时接收不等于 24 小时人工实时响应。
- 页面必须提示急症使用正式急救/就医渠道。
- AI 默认只能生成建议，不能直接发送专业回复。
- 阶段一最多做入口/原型，阶段二才作为 Must 闭环。

## 当前进展

- 已在需求 R2 中完成状态机和阶段归属消歧。

## 后续待办

- 甲方确认值班、SLA、升级和电话/录音政策。

## 相关需求

- [满意度评价](sleep-care-satisfaction.md)
- [AI 与审核知识](sleep-care-ai-assist.md)

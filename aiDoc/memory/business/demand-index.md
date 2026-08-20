# 业务需求索引

## Active

> 以下条目均作为 R2 第 16 节的工程实施基线执行；涉及真实数据、医疗内容、正式通知或 AI 的能力，仍须通过对应启用门禁。

- [睡眠康养：阶段一集中验收](active/sleep-care-phase1-acceptance.md) — `active` — 收口固定时钟、四角色主流程、失败分支、负向权限和阶段证据。
- [睡眠康养：阶段二集中验收与复盘](active/sleep-care-phase2-acceptance.md) — `active` — 集中验证自动化、迁移幂等、权限、页面证据和真实能力关闭门禁。
- [睡眠康养：本地运行真实时钟](active/sleep-care-real-time-clock.md) — `active` — 日常 Compose 使用系统时间，固定业务时钟仅在可重复验收时显式启用。
- [睡眠康养：固定数据自然化展示](active/sleep-care-fixture-presentation.md) — `active` — 记录编码与名称采用自然业务值，环境边界由页面提示和内部字段独立表达。
- [睡眠康养：阶段一通俗化与流程简化](active/sleep-care-plain-language-flow.md) — `active` — 收敛非技术文案、自动转交和账号切换后的角色首页导航。
- [睡眠康养：阶段一 OSA 展示闭环](active/sleep-care-phase1-demo.md) — `active` — 用固定夹具走通康养用户提交、关注事项、医护处置和上级督导。
- [睡眠康养：用户公共资料与授权](active/sleep-care-client-consent.md) — `active` — 建立唯一公共资料、服务授权和责任关系历史。
- [睡眠康养：组织角色与数据权限](active/sleep-care-access-control.md) — `active` — 组合菜单、API、按钮、DataScope、机构和责任关系权限。
- [睡眠康养：OSA 路径与 D1–D5 计划](active/sleep-care-osa-plan.md) — `active` — 建立版本化 OSA 方案、计划实例和任务时间线。
- [睡眠康养：问卷与规则版本](active/sleep-care-questionnaire-versioning.md) — `active` — 发布不可变问卷/规则版本并保留答卷历史。
- [睡眠康养：关注事项闭环](active/sleep-care-attention-case.md) — `active` — 规则命中后完成人工确认、处置、升级、关闭和重开。
- [睡眠康养：员工责任范围工作台](active/sleep-care-staff-workbench.md) — `active` — 聚合职责范围任务与事项，并提供管家、医护分角色动作入口。
- [睡眠康养：通知状态与重试](active/sleep-care-notification.md) — `active` — 区分受理、送达与用户交互，并以默认关闭的 provider 契约控制验签、重试、限流和费用。
- [睡眠康养：主动咨询服务](active/sleep-care-consultation.md) — `active` — 阶段二实现咨询分配、转交、升级、解决和关闭。
- [睡眠康养：每日汇总与督导](active/sleep-care-supervision.md) — `active` — 生成十二项机构级日报、运营覆盖与追加修正，并支持上级指导和介入。
- [睡眠康养：移动端受限访问](active/sleep-care-mobile-access.md) — `active` — 以独立移动布局和受限会话访问本人任务。
- [睡眠康养：满意度评价](active/sleep-care-satisfaction.md) — `active` — 服务闭环后发起评价并保留关联与质量跟进。
- [睡眠康养：AI 与审核知识](active/sleep-care-ai-assist.md) — `active` — 阶段二可选影子验证，阶段三正式受限交付，自动回复另设门禁。
- [睡眠康养：阶段二试用准备与桌面演练](active/sleep-care-trial-rehearsal.md) — `active` — UAT/培训资产、灰度门禁和无副作用暂停/回滚桌面演练。
- [睡眠康养：设备数据](active/sleep-care-device-data.md) — `active` — 先支持人工观察数据，后续以 adapter 接入厂商。
- [睡眠康养：失眠 CBT-I 与睡眠日记](active/sleep-care-insomnia-cbti.md) — `active` — 阶段三建设独立路径，首期只预留多路径模型。

## Done

- [睡眠康养：计划状态显示修复](done/sleep-care-plan-status-display.md) — `done` — 已发布计划按接口生命周期字段正确显示为可使用，并有回归测试保护。

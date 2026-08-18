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
- 两个接口只允许医护和上级医师角色；前端 `CareQuestionnaires` 页面只读，不提供内容发布或答卷提交。
- 答卷写侧是后端 Go 服务边界，P1-03 不注册 HTTP 接口；P1-05 接入前不得从前端绕过该边界。

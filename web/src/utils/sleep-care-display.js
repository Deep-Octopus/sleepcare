const containsInternalWording = (value) => /\[测试\]|合成数据|固定测试|软件验收|流程验证|继续验证|测试计划|测试流程|测试软件|测试\s*OSA|synthetic/i.test(String(value || ''))

const readableOr = (value, fallback) => {
  const text = String(value || '').trim()
  return !text || containsInternalWording(text) ? fallback : text
}

export const readablePlanTitle = (value) => readableOr(value, '五次服务跟进计划')

export const readablePlanPurpose = (value) => readableOr(value, '用于安排五次服务任务并记录完成进度。')

export const readableTaskTitle = (value, dayCode) => readableOr(
  value,
  dayCode === 'D1' ? '服务流程确认' : '服务进度记录'
)

export const readableQuestionnaireTitle = (value) => readableOr(value, '服务流程确认')

export const readableQuestionnairePurpose = (value) => readableOr(
  value,
  '用于确认任务流程是否需要工作人员继续跟进。'
)

export const readableQuestionTitle = (value) => readableOr(
  value,
  '本次填写完成后，是否需要工作人员继续跟进？'
)

export const readableOptionLabel = (value) => {
  const text = String(value || '').trim()
  if (!containsInternalWording(text)) {
    return text || '未命名选项'
  }
  return text.includes('不创建') ? '不需要工作人员跟进' : '需要工作人员继续跟进'
}

export const readableRuleTitle = (value) => readableOr(value, '工作人员跟进规则')

export const readableAttentionReason = (value) => readableOr(
  value,
  '用户在服务流程中选择了需要工作人员继续跟进。'
)

export const readableReviewNote = (value) => readableOr(value, '内容和流程已确认。')

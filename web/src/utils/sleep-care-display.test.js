import assert from 'node:assert/strict'
import test from 'node:test'
import {
  planLifecycleStatusPresentation,
  readableAttentionReason,
  readableOptionLabel,
  readablePlanTitle,
  readableQuestionTitle,
  readableTaskTitle
} from './sleep-care-display.js'

test('计划状态使用接口 lifecycleStatus 字段展示', () => {
  assert.deepEqual(
    planLifecycleStatusPresentation({ lifecycleStatus: 'PUBLISHED' }),
    { label: '可使用', tagType: 'success' }
  )
  assert.deepEqual(
    planLifecycleStatusPresentation({ lifecycleStatus: 'DISABLED', status: 'PUBLISHED' }),
    { label: '已停用', tagType: 'info' }
  )
})

test('内部数据标识不会显示给页面用户', () => {
  assert.equal(readablePlanTitle('测试 OSA 流程验证计划（非医疗内容）'), '五次服务跟进计划')
  assert.equal(readableTaskTitle('D1 测试流程确认任务', 'D1'), '服务流程确认')
  assert.equal(readableQuestionTitle('是否继续完成本次测试流程验证？'), '本次填写完成后，是否需要工作人员继续跟进？')
  assert.equal(readableOptionLabel('继续验证，不创建人工关注流程'), '不需要工作人员跟进')
  assert.equal(readableAttentionReason('测试流程选项请求创建人工关注链'), '用户在服务流程中选择了需要工作人员继续跟进。')
})

test('普通业务文案保持不变', () => {
  assert.equal(readablePlanTitle('居家关怀计划'), '居家关怀计划')
  assert.equal(readableTaskTitle('联系确认', 'D2'), '联系确认')
  assert.equal(readableOptionLabel('需要工作人员联系'), '需要工作人员联系')
})

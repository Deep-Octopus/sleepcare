import service from '@/utils/request'

const commandHeaders = (idempotencyKey) => ({
  'Idempotency-Key': idempotencyKey || crypto.randomUUID()
})

export const getCareWorkbench = () =>
  service({
    url: '/care/workbench',
    method: 'get'
  })

export const getAttentionCases = (params) =>
  service({
    url: '/care/attention-cases',
    method: 'get',
    params
  })

export const getAttentionCase = (id) =>
  service({
    url: `/care/attention-cases/${id}`,
    method: 'get'
  })

export const acknowledgeAttentionCase = (id, data, idempotencyKey) =>
  service({
    url: `/care/attention-cases/${id}/acknowledge`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

export const createAttentionHandlingRecord = (id, data, idempotencyKey) =>
  service({
    url: `/care/attention-cases/${id}/handling-records`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

export const escalateAttentionCase = (id, data, idempotencyKey) =>
  service({
    url: `/care/attention-cases/${id}/escalate`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

export const closeAttentionCase = (id, data, idempotencyKey) =>
  service({
    url: `/care/attention-cases/${id}/close`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

export const reopenAttentionCase = (id, data, idempotencyKey) =>
  service({
    url: `/care/attention-cases/${id}/reopen`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

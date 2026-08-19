import service from '@/utils/request'

const commandHeaders = (idempotencyKey) => ({
  'Idempotency-Key': idempotencyKey || crypto.randomUUID()
})

export const getSatisfactionResponses = (params) => service({
  url: '/care/satisfaction-responses',
  method: 'get',
  params
})

export const getSatisfactionFollowUps = (params) => service({
  url: '/care/satisfaction-follow-ups',
  method: 'get',
  params
})

export const getSatisfactionFollowUp = (id) => service({
  url: `/care/satisfaction-follow-ups/${id}`,
  method: 'get'
})

export const acknowledgeSatisfactionFollowUp = (id, data, idempotencyKey) => service({
  url: `/care/satisfaction-follow-ups/${id}/acknowledge`,
  method: 'post',
  headers: commandHeaders(idempotencyKey),
  data
})

export const resolveSatisfactionFollowUp = (id, data, idempotencyKey) => service({
  url: `/care/satisfaction-follow-ups/${id}/resolve`,
  method: 'post',
  headers: commandHeaders(idempotencyKey),
  data
})

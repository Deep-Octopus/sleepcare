import service from '@/utils/request'

const commandHeaders = (idempotencyKey) => ({
  'Idempotency-Key': idempotencyKey || crypto.randomUUID()
})

export const getConsultations = (params) => service({
  url: '/care/consultations',
  method: 'get',
  params
})

export const getConsultation = (id) => service({
  url: `/care/consultations/${id}`,
  method: 'get'
})

export const getConsultationAssigneeOptions = (id) => service({
  url: `/care/consultations/${id}/assignee-options`,
  method: 'get'
})

export const assignConsultation = (id, data, idempotencyKey) => service({
  url: `/care/consultations/${id}/assign`,
  method: 'post',
  data,
  headers: commandHeaders(idempotencyKey)
})

export const replyConsultation = (id, data, idempotencyKey) => service({
  url: `/care/consultations/${id}/replies`,
  method: 'post',
  data,
  headers: commandHeaders(idempotencyKey)
})

export const transferConsultation = (id, data, idempotencyKey) => service({
  url: `/care/consultations/${id}/transfer`,
  method: 'post',
  data,
  headers: commandHeaders(idempotencyKey)
})

export const escalateConsultation = (id, data, idempotencyKey) => service({
  url: `/care/consultations/${id}/escalate`,
  method: 'post',
  data,
  headers: commandHeaders(idempotencyKey)
})

export const resolveConsultation = (id, data, idempotencyKey) => service({
  url: `/care/consultations/${id}/resolve`,
  method: 'post',
  data,
  headers: commandHeaders(idempotencyKey)
})

export const closeConsultation = (id, data, idempotencyKey) => service({
  url: `/care/consultations/${id}/close`,
  method: 'post',
  data,
  headers: commandHeaders(idempotencyKey)
})

export const reopenConsultation = (id, data, idempotencyKey) => service({
  url: `/care/consultations/${id}/reopen`,
  method: 'post',
  data,
  headers: commandHeaders(idempotencyKey)
})

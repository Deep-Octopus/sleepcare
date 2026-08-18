import service from '@/utils/request'

const commandHeaders = (idempotencyKey) => ({
  'Idempotency-Key': idempotencyKey || crypto.randomUUID()
})

export const getDailySummaries = (params) =>
  service({
    url: '/care/daily-summaries',
    method: 'get',
    params
  })

export const getDailySummary = (id) =>
  service({
    url: `/care/daily-summaries/${id}`,
    method: 'get'
  })

export const getSupervisionReviews = (params) =>
  service({
    url: '/care/reviews',
    method: 'get',
    params
  })

export const addSupervisorGuidance = (id, data, idempotencyKey) =>
  service({
    url: `/care/reviews/${id}/guidance`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

export const interveneSupervisionReview = (id, data, idempotencyKey) =>
  service({
    url: `/care/reviews/${id}/intervene`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

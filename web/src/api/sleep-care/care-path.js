import service from '@/utils/request'

const commandHeaders = (idempotencyKey) => ({
  'Idempotency-Key': idempotencyKey || crypto.randomUUID()
})

export const getPlanVersions = (params) =>
  service({
    url: '/care/plan-versions',
    method: 'get',
    params
  })

export const getPlanVersion = (id) =>
  service({
    url: `/care/plan-versions/${id}`,
    method: 'get'
  })

export const previewCarePlan = (careClientId, data, idempotencyKey) =>
  service({
    url: `/care/clients/${careClientId}/plan-previews`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

export const startCarePlan = (careClientId, data, idempotencyKey) =>
  service({
    url: `/care/clients/${careClientId}/plan-instances`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

export const getClientPlans = (careClientId) =>
  service({
    url: `/care/clients/${careClientId}/plan-instances`,
    method: 'get'
  })

export const pauseCarePlan = (planInstanceId, data, idempotencyKey) =>
  service({
    url: `/care/plan-instances/${planInstanceId}/pause`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

export const resumeCarePlan = (planInstanceId, data, idempotencyKey) =>
  service({
    url: `/care/plan-instances/${planInstanceId}/resume`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

export const getCareTasks = (params) =>
  service({
    url: '/care/tasks',
    method: 'get',
    params
  })

export const getCareTask = (id) =>
  service({
    url: `/care/tasks/${id}`,
    method: 'get'
  })

export const recordCareTaskContact = (id, data, idempotencyKey) =>
  service({
    url: `/care/tasks/${id}/contact-records`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

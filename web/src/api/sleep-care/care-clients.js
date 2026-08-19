import service from '@/utils/request'

const idempotencyHeaders = () => ({
  'Idempotency-Key': crypto.randomUUID()
})

export const getCareClients = (params) =>
  service({ url: '/care/clients', method: 'get', params })

export const getCareClient = (id) =>
  service({ url: `/care/clients/${id}`, method: 'get' })

export const getCareClientOptions = () =>
  service({ url: '/care/client-options', method: 'get' })

export const getDataGovernanceReadiness = () =>
  service({ url: '/care/data-governance-readiness', method: 'get' })

export const getDataLifecycleRequests = (id, params) =>
  service({
    url: `/care/clients/${id}/data-lifecycle-requests`,
    method: 'get',
    params
  })

export const createCareClient = (data) =>
  service({
    url: '/care/clients',
    method: 'post',
    data,
    headers: idempotencyHeaders()
  })

export const updateCareClient = (id, data) =>
  service({
    url: `/care/clients/${id}`,
    method: 'put',
    data,
    headers: idempotencyHeaders()
  })

export const createCareAssignment = (id, data) =>
  service({
    url: `/care/clients/${id}/assignments`,
    method: 'post',
    data,
    headers: idempotencyHeaders()
  })

export const createCareConsent = (id, data) =>
  service({
    url: `/care/clients/${id}/consent-records`,
    method: 'post',
    data,
    headers: idempotencyHeaders()
  })

export const createDataLifecycleRequest = (id, data, idempotencyKey) =>
  service({
    url: `/care/clients/${id}/data-lifecycle-requests`,
    method: 'post',
    data,
    headers: {
      'Idempotency-Key': idempotencyKey
    }
  })

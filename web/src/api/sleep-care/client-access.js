import service from '@/utils/request'

const clientRequest = (config) => service({
  authContext: 'client',
  silentError: true,
  ...config
})

export const redeemClientAccess = (grant) => clientRequest({
  url: '/care/client-access/redeem',
  method: 'post',
  data: { grant }
})

export const getClientTasks = (params = {}) => clientRequest({
  url: '/care/client/tasks',
  method: 'get',
  params
})

export const getClientTask = (taskId) => clientRequest({
  url: `/care/client/tasks/${taskId}`,
  method: 'get'
})

export const getClientQuestionnaire = (taskId) => clientRequest({
  url: `/care/client/tasks/${taskId}/questionnaire`,
  method: 'get'
})

export const recordClientInteraction = (taskId, idempotencyKey, data) => clientRequest({
  url: `/care/client/tasks/${taskId}/interactions`,
  method: 'post',
  headers: {
    'Idempotency-Key': idempotencyKey
  },
  data
})

export const saveClientDraft = (taskId, idempotencyKey, data) => clientRequest({
  url: `/care/client/tasks/${taskId}/draft`,
  method: 'put',
  headers: {
    'Idempotency-Key': idempotencyKey
  },
  data
})

export const submitClientTask = (taskId, idempotencyKey, data) => clientRequest({
  url: `/care/client/tasks/${taskId}/submit`,
  method: 'post',
  headers: {
    'Idempotency-Key': idempotencyKey
  },
  data
})

export const getClientConsultations = (params = {}) => clientRequest({
  url: '/care/client/consultations',
  method: 'get',
  params
})

export const getClientConsultation = (id) => clientRequest({
  url: `/care/client/consultations/${id}`,
  method: 'get'
})

export const createClientConsultation = (idempotencyKey, data) => clientRequest({
  url: '/care/client/consultations',
  method: 'post',
  headers: {
    'Idempotency-Key': idempotencyKey
  },
  data
})

export const addClientConsultationMessage = (id, idempotencyKey, data) => clientRequest({
  url: `/care/client/consultations/${id}/messages`,
  method: 'post',
  headers: {
    'Idempotency-Key': idempotencyKey
  },
  data
})

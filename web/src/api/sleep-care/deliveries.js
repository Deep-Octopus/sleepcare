import service from '@/utils/request'

const commandHeaders = (idempotencyKey) => ({
  'Idempotency-Key': idempotencyKey || crypto.randomUUID()
})

export const getCareDeliveries = (params) =>
  service({
    url: '/care/deliveries',
    method: 'get',
    params
  })

export const getNotificationProviderReadiness = () =>
  service({
    url: '/care/notification-provider-readiness',
    method: 'get'
  })

export const resendCareDelivery = (id, data, idempotencyKey) =>
  service({
    url: `/care/deliveries/${id}/resend`,
    method: 'post',
    data,
    headers: commandHeaders(idempotencyKey)
  })

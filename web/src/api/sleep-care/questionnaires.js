import service from '@/utils/request'

export const getQuestionnaireVersions = (params) =>
  service({
    url: '/care/questionnaire-versions',
    method: 'get',
    params
  })

export const getQuestionnaireVersion = (id) =>
  service({
    url: `/care/questionnaire-versions/${id}`,
    method: 'get'
  })

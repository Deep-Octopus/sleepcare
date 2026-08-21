import { clearClientDraftState } from '@/utils/client-session'

export const unwrapClientResponse = (response) => {
  if (response?.code === 0) {
    return response.data
  }
  const error = new Error(response?.msg || '请求未完成')
  error.code = response?.code
  throw error
}

export const newIdempotencyKey = () => {
  if (globalThis.crypto?.randomUUID) {
    return globalThis.crypto.randomUUID()
  }
  return `client-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

const localDraftKey = (taskId) => `gva-client-task-draft:${taskId}`
const submitKey = (taskId) => `gva-client-task-submit-key:${taskId}`
const satisfactionSubmitKey = (requestId) => `gva-client-satisfaction-submit-key:${requestId}`

export const compactAnswers = (answers = {}) => Object.fromEntries(
  Object.entries(answers).filter(([, value]) => (
    value !== undefined &&
    value !== null &&
    value !== '' &&
    (!Array.isArray(value) || value.length > 0)
  ))
)

export const readLocalDraft = (taskId) => {
  try {
    const raw = localStorage.getItem(localDraftKey(taskId))
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export const writeLocalDraft = (taskId, answers) => {
  localStorage.setItem(localDraftKey(taskId), JSON.stringify({
    answers: compactAnswers(answers),
    savedAt: new Date().toISOString()
  }))
  localStorage.removeItem(submitKey(taskId))
}

export const clearLocalTaskState = (taskId) => {
  localStorage.removeItem(localDraftKey(taskId))
  localStorage.removeItem(submitKey(taskId))
}

export const clearAllClientLocalState = () => {
  clearClientDraftState()
}

export const getOrCreateSubmitKey = (taskId) => {
  const existing = localStorage.getItem(submitKey(taskId))
  if (existing) {
    return existing
  }
  const value = newIdempotencyKey()
  localStorage.setItem(submitKey(taskId), value)
  return value
}

export const getOrCreateSatisfactionSubmitKey = (requestId) => {
  const key = satisfactionSubmitKey(requestId)
  const existing = localStorage.getItem(key)
  if (existing) {
    return existing
  }
  const value = newIdempotencyKey()
  localStorage.setItem(key, value)
  return value
}

export const clearSatisfactionSubmitKey = (requestId) => {
  localStorage.removeItem(satisfactionSubmitKey(requestId))
}

export const formatTaskTime = (value) => {
  if (!value) {
    return '暂无'
  }
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value))
}

export const formatTaskWindow = (openAt, dueAt) => {
  if (!openAt || !dueAt) {
    return `${formatTaskTime(openAt)} - ${formatTaskTime(dueAt)}`
  }

  const openDate = new Date(openAt)
  const dueDate = new Date(dueAt)
  const sameDay = openDate.getFullYear() === dueDate.getFullYear() &&
    openDate.getMonth() === dueDate.getMonth() &&
    openDate.getDate() === dueDate.getDate()

  if (!sameDay) {
    return `${formatTaskTime(openAt)} - ${formatTaskTime(dueAt)}`
  }

  const dateLabel = new Intl.DateTimeFormat('zh-CN', {
    month: 'long',
    day: 'numeric'
  }).format(openDate)
  const timeFormatter = new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit'
  })
  return `${dateLabel} ${timeFormatter.format(openDate)} - ${timeFormatter.format(dueDate)}`
}

export const taskStateCopy = (task) => {
  if (task.executionStatus === 'SUBMITTED') {
    return { label: '已提交', tone: 'success' }
  }
  if (task.executionStatus === 'CANCELLED') {
    return { label: '已取消', tone: 'muted' }
  }
  if (task.timingStatus === 'NOT_OPEN') {
    return { label: '尚未开放', tone: 'muted' }
  }
  if (task.timingStatus === 'EXPIRED') {
    return { label: '已结束', tone: 'danger' }
  }
  if (task.executionStatus === 'IN_PROGRESS') {
    return { label: '继续填写', tone: 'active' }
  }
  return { label: '可以填写', tone: 'active' }
}

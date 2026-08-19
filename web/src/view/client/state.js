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
  const prefixes = ['gva-client-task-draft:', 'gva-client-task-submit-key:']
  Object.keys(localStorage).forEach((key) => {
    if (prefixes.some((prefix) => key.startsWith(prefix))) {
      localStorage.removeItem(key)
    }
  })
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

export const formatTaskTime = (value) => {
  if (!value) {
    return '—'
  }
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value))
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

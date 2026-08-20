const CLIENT_AUTH_MODE_KEY = 'gva-client-auth-mode'

export const CLIENT_AUTH_MODE_ACCOUNT = 'account'
export const CLIENT_AUTH_MODE_GRANT = 'grant'

export const readClientAuthMode = () => {
  if (typeof window === 'undefined') {
    return ''
  }
  return sessionStorage.getItem(CLIENT_AUTH_MODE_KEY) || ''
}

export const writeClientAuthMode = (mode) => {
  if (typeof window === 'undefined') {
    return
  }
  sessionStorage.setItem(CLIENT_AUTH_MODE_KEY, mode)
}

export const clearClientAuthMode = () => {
  if (typeof window === 'undefined') {
    return
  }
  sessionStorage.removeItem(CLIENT_AUTH_MODE_KEY)
}

export const clearClientDraftState = () => {
  if (typeof window === 'undefined') {
    return
  }
  const prefixes = [
    'gva-client-task-draft:',
    'gva-client-task-submit-key:',
    'gva-client-satisfaction-submit-key:'
  ]
  Object.keys(localStorage).forEach((key) => {
    if (prefixes.some((prefix) => key.startsWith(prefix))) {
      localStorage.removeItem(key)
    }
  })
}

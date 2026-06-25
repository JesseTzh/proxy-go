type ApiUiState = {
  pendingCount: number
  error: string | null
}

type ApiUiListener = (state: ApiUiState) => void

let state: ApiUiState = { pendingCount: 0, error: null }
const listeners = new Set<ApiUiListener>()

function emit() {
  for (const listener of listeners) listener(state)
}

export function subscribeApiUi(listener: ApiUiListener) {
  listeners.add(listener)
  listener(state)
  return () => listeners.delete(listener)
}

export function getApiUiState() {
  return state
}

export function beginApiRequest(trackLoading: boolean) {
  if (!trackLoading) return
  state = { ...state, pendingCount: state.pendingCount + 1 }
  emit()
}

export function endApiRequest(trackLoading: boolean) {
  if (!trackLoading) return
  state = { ...state, pendingCount: Math.max(0, state.pendingCount - 1) }
  emit()
}

export function showApiError(message: string) {
  state = { ...state, error: message }
  emit()
}

export function clearApiError() {
  state = { ...state, error: null }
  emit()
}

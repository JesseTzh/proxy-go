import { lazy, Suspense, useEffect, useState } from 'react'
import { Toaster } from 'sonner'
import { ApiFeedback } from '../components/ApiFeedback'
import { MockDebugPanel } from '../components/MockDebugPanel'
import { getJson, postJson } from '../lib/api'
import { isMockMode } from '../lib/mockApi'

const LoginPage = lazy(() => import('../pages/LoginPage').then(module => ({ default: module.LoginPage })))
const AuthenticatedApp = lazy(() => import('./AuthenticatedApp').then(module => ({ default: module.AuthenticatedApp })))

export function App() {
  const [authed, setAuthed] = useState(false)
  const [checking, setChecking] = useState(true)
  useEffect(() => { getJson('auth/me', { silentError: true }).then(()=>setAuthed(true)).catch(()=>setAuthed(false)).finally(()=>setChecking(false)) }, [])
  useEffect(() => {
    if (!isMockMode) return
    const syncAuth = (event: Event) => setAuthed((event as CustomEvent<boolean>).detail)
    window.addEventListener('proxy-go:mock-auth', syncAuth)
    return () => window.removeEventListener('proxy-go:mock-auth', syncAuth)
  }, [])

  async function logout() {
    await postJson('auth/logout')
    setAuthed(false)
  }

  if (checking) return <div className="p-8" data-testid="app-loading">加载中…</div>
  return (
    <>
      <Suspense fallback={<AppRouteLoading />}>
        {authed ? <AuthenticatedApp onLogout={logout} /> : <LoginPage onLogin={() => setAuthed(true)} />}
      </Suspense>
      <Toaster richColors />
      <ApiFeedback />
      <MockDebugPanel />
    </>
  )
}

function AppRouteLoading() {
  return <div className="p-8" data-testid="app-route-loading">加载中…</div>
}

import { lazy, Suspense, useEffect, useState } from 'react'
import { Toaster } from 'sonner'
import { ApiFeedback } from '../components/ApiFeedback'
import { getJson } from '../lib/api'

const LoginPage = lazy(() => import('../pages/LoginPage').then(module => ({ default: module.LoginPage })))
const AuthenticatedApp = lazy(() => import('./AuthenticatedApp').then(module => ({ default: module.AuthenticatedApp })))

export function App() {
  const [authed, setAuthed] = useState(false)
  const [checking, setChecking] = useState(true)
  useEffect(() => { getJson('auth/me', { silentError: true }).then(()=>setAuthed(true)).catch(()=>setAuthed(false)).finally(()=>setChecking(false)) }, [])
  if (checking) return <div className="p-8" data-testid="app-loading">加载中…</div>
  return (
    <>
      <Suspense fallback={<AppRouteLoading />}>
        {authed ? <AuthenticatedApp /> : <LoginPage onLogin={() => setAuthed(true)} />}
      </Suspense>
      <Toaster richColors />
      <ApiFeedback />
    </>
  )
}

function AppRouteLoading() {
  return <div className="p-8" data-testid="app-route-loading">加载中…</div>
}

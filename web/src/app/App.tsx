import { useEffect, useState } from 'react'
import { getJson } from '../lib/api'
import { Shell } from './Shell'
import type { Page } from './navigation'
import { AuditPage } from '../pages/AuditPage'
import { DashboardPage } from '../pages/DashboardPage'
import { DomainsPage } from '../pages/DomainsPage'
import { LoginPage } from '../pages/LoginPage'
import { ReverseProxiesPage } from '../pages/ReverseProxiesPage'
import { SettingsPage } from '../pages/SettingsPage'
import { VlessPage } from '../pages/VlessPage'

export function App() {
  const [authed, setAuthed] = useState(false)
  const [checking, setChecking] = useState(true)
  const [page, setPage] = useState<Page>('dashboard')
  useEffect(() => { getJson('auth/me').then(()=>setAuthed(true)).catch(()=>setAuthed(false)).finally(()=>setChecking(false)) }, [])
  if (checking) return <div className="p-8" data-testid="app-loading">加载中…</div>
  if (!authed) return <LoginPage onLogin={() => setAuthed(true)} />
  return <Shell page={page} setPage={setPage}><PageContent page={page} /></Shell>
}

function PageContent({page}:{page:Page}){
  switch(page){
    case 'dashboard': return <DashboardPage/>
    case 'domains': return <DomainsPage/>
    case 'reverse': return <ReverseProxiesPage/>
    case 'vless': return <VlessPage/>
    case 'settings': return <SettingsPage/>
    case 'audit': return <AuditPage/>
  }
}

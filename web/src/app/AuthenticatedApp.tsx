import { lazy, Suspense, useState } from 'react'
import { Shell } from './Shell'
import type { Page } from './navigation'
import { DashboardPage } from '../pages/DashboardPage'

const DomainsPage = lazy(() => import('../pages/DomainsPage').then(module => ({ default: module.DomainsPage })))
const ReverseProxiesPage = lazy(() => import('../pages/ReverseProxiesPage').then(module => ({ default: module.ReverseProxiesPage })))
const VlessPage = lazy(() => import('../pages/VlessPage').then(module => ({ default: module.VlessPage })))
const WireGuardPage = lazy(() => import('../pages/WireGuardPage').then(module => ({ default: module.WireGuardPage })))
const SettingsPage = lazy(() => import('../pages/SettingsPage').then(module => ({ default: module.SettingsPage })))
const AuditPage = lazy(() => import('../pages/AuditPage').then(module => ({ default: module.AuditPage })))

export function AuthenticatedApp({ onLogout }: { onLogout: () => Promise<void> }) {
  const [page, setPage] = useState<Page>('dashboard')

  return (
    <Shell page={page} setPage={setPage}>
      <Suspense fallback={<PageLoading />}>
        <PageContent page={page} onLogout={onLogout} />
      </Suspense>
    </Shell>
  )
}

function PageContent({ page, onLogout }: { page: Page; onLogout: () => Promise<void> }) {
  switch (page) {
    case 'dashboard': return <DashboardPage onLogout={onLogout} />
    case 'domains': return <DomainsPage />
    case 'reverse': return <ReverseProxiesPage />
    case 'vless': return <VlessPage />
    case 'wireguard': return <WireGuardPage />
    case 'settings': return <SettingsPage />
    case 'audit': return <AuditPage />
  }
}

function PageLoading() {
  return <div className="p-4 text-sm text-muted-foreground" data-testid="page-loading">加载中…</div>
}

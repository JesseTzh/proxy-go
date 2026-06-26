import { useEffect, useState } from 'react'
import { FileJson, Settings } from 'lucide-react'
import { toast } from 'sonner'
import { JsonPreview } from '../components/JsonPreview'
import { PageHeader } from '../components/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { getJson, postJson } from '../lib/api'
import type { RuntimeStatus } from '../types'

type Tab = 'system' | 'runtime'

export function SettingsPage() {
  const [tab, setTab] = useState<Tab>('system')
  const [data, setData] = useState<unknown>()
  const [runtime, setRuntime] = useState<RuntimeStatus>()

  useEffect(() => {
    void getJson('settings').then(setData)
    void getJson<RuntimeStatus>('runtime/status').then(setRuntime)
  }, [])

  return (
    <div className="space-y-4" data-testid="settings-page">
      <PageHeader title="系统设置" desc="ACME 邮箱、初始端口状态、运行目录、版本信息与完整 Runtime 状态。" data-testid="settings-header" />

      <div className="inline-flex rounded-lg bg-white p-1 shadow-[var(--shadow-border-subtle)]" data-testid="settings-tabs">
        <TabButton active={tab === 'system'} onClick={() => setTab('system')} data-testid="settings-system-tab">
          <Settings size={16} aria-hidden="true" />
          系统信息
        </TabButton>
        <TabButton active={tab === 'runtime'} onClick={() => setTab('runtime')} data-testid="settings-runtime-tab">
          <FileJson size={16} aria-hidden="true" />
          Runtime JSON
        </TabButton>
      </div>

      {tab === 'system' ? (
        <section className="space-y-4" data-testid="settings-system-panel">
          <JsonPreview value={data ?? {}} data-testid="settings-system-json" />
          <Button variant="outline" onClick={() => postJson('init/disable-initial-port').then(() => toast.success('已关闭，重启后生效'))} data-testid="settings-disable-initial-port">
            关闭初始管理端口
          </Button>
        </section>
      ) : (
        <section className="space-y-3" data-testid="settings-runtime-panel">
          <Card>
            <CardHeader>
              <CardTitle>完整 Runtime 状态</CardTitle>
              <CardDescription>用于排查接口返回、进程状态和资源计数，Runtime 页面不再直接展示此 JSON。</CardDescription>
            </CardHeader>
          </Card>
          <JsonPreview value={runtime ?? {}} data-testid="settings-runtime-json" />
        </section>
      )}
    </div>
  )
}

function TabButton({
  active,
  children,
  onClick,
  'data-testid': dataTestId,
}: {
  active: boolean
  children: React.ReactNode
  onClick: () => void
  'data-testid'?: string
}) {
  return (
    <button
      type="button"
      className={`inline-flex items-center gap-2 rounded-md px-3 py-2 text-sm ${active ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted'}`}
      onClick={onClick}
      data-testid={dataTestId}
    >
      {children}
    </button>
  )
}

import { useEffect, useState } from 'react'
import { Download, Power, PowerOff, Save, Trash2, UserPlus } from 'lucide-react'
import { toast } from 'sonner'
import { DataTable } from '../components/DataTable'
import { FormField } from '../components/FormField'
import { PageHeader } from '../components/PageHeader'
import { StatusBadge } from '../components/StatusBadge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { TableCell, TableRow } from '@/components/ui/table'
import { delJson, getJson, postJson, putJson } from '../lib/api'
import type { WireGuardClient, WireGuardServer, WireGuardState } from '../types'

const defaultServer: WireGuardServer = {
  id: 1,
  enabled: false,
  interfaceName: 'wg0',
  address: '10.8.0.1/24',
  listenPort: 51820,
  dns: '1.1.1.1',
  mtu: 1420,
  egressInterface: 'eth0',
  publicKey: '',
}

export function WireGuardPage() {
  const [state, setState] = useState<WireGuardState>()
  const [server, setServer] = useState<WireGuardServer>(defaultServer)
  const [clientName, setClientName] = useState('')
  const [busy, setBusy] = useState<string>()

  async function load() {
    const next = await getJson<WireGuardState>('wireguard')
    setState(next)
    setServer(next.server)
  }

  useEffect(() => { void load() }, [])

  async function saveServer() {
    setBusy('save')
    try {
      await putJson('wireguard', {
        enabled: server.enabled,
        address: server.address,
        listenPort: Number(server.listenPort),
        dns: server.dns,
        mtu: Number(server.mtu),
        egressInterface: server.egressInterface,
      })
      toast.success(server.enabled ? 'WireGuard 配置已应用' : 'WireGuard 已停用')
      await load()
    } finally {
      setBusy(undefined)
    }
  }

  async function createClient(event: React.FormEvent) {
    event.preventDefault()
    setBusy('create')
    try {
      await postJson('wireguard/clients', { name: clientName })
      setClientName('')
      toast.success('客户端已创建')
      await load()
    } finally {
      setBusy(undefined)
    }
  }

  async function clientAction(client: WireGuardClient, action: 'enable' | 'disable' | 'delete') {
    setBusy(`${client.id}-${action}`)
    try {
      if (action === 'delete') await delJson(`wireguard/clients/${client.id}`)
      else await postJson(`wireguard/clients/${client.id}/${action}`)
      toast.success(action === 'delete' ? '客户端已删除' : `客户端已${action === 'enable' ? '启用' : '停用'}`)
      await load()
    } finally {
      setBusy(undefined)
    }
  }

  function updateServer<K extends keyof WireGuardServer>(key: K, value: WireGuardServer[K]) {
    setServer(current => ({ ...current, [key]: value }))
  }

  return (
    <div className="space-y-4" data-testid="wireguard-page">
      <PageHeader title="WireGuard" desc="管理 VPN 服务、出口域名与客户端配置。" data-testid="wireguard-header" />

      <section className="grid gap-4 xl:grid-cols-[2fr_1fr]" data-testid="wireguard-server-section">
        <Card data-testid="wireguard-settings-card">
          <CardHeader data-testid="wireguard-settings-header">
            <CardTitle data-testid="wireguard-settings-title">服务配置</CardTitle>
            <CardDescription data-testid="wireguard-settings-description">配置变更保存后会自动重建 WireGuard 接口。</CardDescription>
          </CardHeader>
          <CardContent data-testid="wireguard-settings-content">
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3" data-testid="wireguard-settings-grid">
              <FormField label="服务网段" description="服务端在 VPN 网段中的地址，例如 10.8.0.1/24。" data-testid="wireguard-address-field">
                <Input value={server.address} onChange={event => updateServer('address', event.target.value)} data-testid="wireguard-address-input" />
              </FormField>
              <FormField label="UDP 端口" data-testid="wireguard-port-field">
                <Input type="number" value={server.listenPort} onChange={event => updateServer('listenPort', Number(event.target.value))} data-testid="wireguard-port-input" />
              </FormField>
              <FormField label="客户端 DNS" data-testid="wireguard-dns-field">
                <Input value={server.dns} onChange={event => updateServer('dns', event.target.value)} data-testid="wireguard-dns-input" />
              </FormField>
              <FormField label="MTU" data-testid="wireguard-mtu-field">
                <Input type="number" value={server.mtu} onChange={event => updateServer('mtu', Number(event.target.value))} data-testid="wireguard-mtu-input" />
              </FormField>
              <FormField label="公网网卡" description="NAT 出口网卡，Docker 默认是 eth0。" data-testid="wireguard-egress-field">
                <Input value={server.egressInterface} onChange={event => updateServer('egressInterface', event.target.value)} data-testid="wireguard-egress-input" />
              </FormField>
              <div className="grid content-end gap-2" data-testid="wireguard-enabled-field">
                <label className="flex h-8 items-center gap-2 text-sm" data-testid="wireguard-enabled-label">
                  <Checkbox checked={server.enabled} onCheckedChange={checked => updateServer('enabled', checked === true)} data-testid="wireguard-enabled-checkbox" />
                  启用 WireGuard 服务
                </label>
                <Button onClick={saveServer} disabled={busy === 'save'} data-testid="wireguard-save-button">
                  <Save aria-hidden="true" />
                  {busy === 'save' ? '应用中…' : '保存并应用'}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card data-testid="wireguard-status-card">
          <CardHeader data-testid="wireguard-status-header">
            <CardTitle data-testid="wireguard-status-title">服务状态</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3" data-testid="wireguard-status-content">
            <div className="flex items-center justify-between gap-3" data-testid="wireguard-runtime-row">
              <span data-testid="wireguard-runtime-label">运行状态</span>
              <StatusBadge tone={state?.runtime.running ? 'success' : 'neutral'}>{state?.runtime.running ? '运行中' : '未运行'}</StatusBadge>
            </div>
            <div className="flex items-center justify-between gap-3" data-testid="wireguard-domain-row">
              <span data-testid="wireguard-domain-label">出口域名</span>
              <span className="break-all text-right font-medium" data-testid="wireguard-domain-value">{server.domain?.domain ?? '未配置'}</span>
            </div>
            <div className="flex items-center justify-between gap-3" data-testid="wireguard-endpoint-row">
              <span data-testid="wireguard-endpoint-label">客户端 Endpoint</span>
              <span className="break-all text-right font-mono text-xs" data-testid="wireguard-endpoint-value">{server.domain ? `${server.domain.domain}:${server.listenPort}` : '-'}</span>
            </div>
            <div className="flex items-center justify-between gap-3" data-testid="wireguard-public-key-row">
              <span data-testid="wireguard-public-key-label">服务端公钥</span>
              <span className="max-w-52 truncate font-mono text-xs" title={server.publicKey} data-testid="wireguard-public-key-value">{server.publicKey || '-'}</span>
            </div>
          </CardContent>
        </Card>
      </section>

      <Card className="p-4" data-testid="wireguard-client-create-card">
        <form className="grid gap-3 md:grid-cols-[1fr_auto] md:items-end" onSubmit={createClient} data-testid="wireguard-client-create-form">
          <FormField label="客户端名称" data-testid="wireguard-client-name-field">
            <Input value={clientName} onChange={event => setClientName(event.target.value)} placeholder="MacBook Pro" data-testid="wireguard-client-name-input" />
          </FormField>
          <Button type="submit" disabled={busy === 'create' || !clientName.trim()} data-testid="wireguard-client-create-button">
            <UserPlus aria-hidden="true" />
            新增客户端
          </Button>
        </form>
      </Card>

      <DataTable headers={['客户端', 'VPN 地址', '状态', '创建时间', '操作']} data-testid="wireguard-clients-table">
        {(state?.clients ?? []).map(client => (
          <TableRow key={client.id} data-testid={`wireguard-client-row-${client.id}`}>
            <TableCell data-testid={`wireguard-client-name-${client.id}`}>{client.name}</TableCell>
            <TableCell className="font-mono text-xs" data-testid={`wireguard-client-address-${client.id}`}>{client.address}</TableCell>
            <TableCell data-testid={`wireguard-client-status-${client.id}`}><StatusBadge tone={client.enabled ? 'success' : 'neutral'}>{client.enabled ? '启用' : '停用'}</StatusBadge></TableCell>
            <TableCell data-testid={`wireguard-client-created-${client.id}`}>{new Date(client.createdAt).toLocaleString()}</TableCell>
            <TableCell data-testid={`wireguard-client-actions-${client.id}`}>
              <div className="flex flex-wrap gap-2" data-testid={`wireguard-client-actions-wrap-${client.id}`}>
                <Button variant="outline" size="sm" onClick={() => { window.location.href = `/api/wireguard/clients/${client.id}/config` }} data-testid={`wireguard-client-download-${client.id}`}>
                  <Download aria-hidden="true" />
                  下载配置
                </Button>
                <Button variant="outline" size="sm" disabled={busy === `${client.id}-${client.enabled ? 'disable' : 'enable'}`} onClick={() => clientAction(client, client.enabled ? 'disable' : 'enable')} data-testid={`wireguard-client-toggle-${client.id}`}>
                  {client.enabled ? <PowerOff aria-hidden="true" /> : <Power aria-hidden="true" />}
                  {client.enabled ? '停用' : '启用'}
                </Button>
                <Button variant="destructive" size="sm" disabled={busy === `${client.id}-delete`} onClick={() => clientAction(client, 'delete')} data-testid={`wireguard-client-delete-${client.id}`}>
                  <Trash2 aria-hidden="true" />
                  删除
                </Button>
              </div>
            </TableCell>
          </TableRow>
        ))}
      </DataTable>
    </div>
  )
}

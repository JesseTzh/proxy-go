import { lazy, Suspense, useEffect, useMemo, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm } from 'react-hook-form'
import { Code2, Copy, Edit, Plus, Power, PowerOff, QrCode, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { DataTable } from '../components/DataTable'
import { FormField } from '../components/FormField'
import { PageHeader } from '../components/PageHeader'
import { StatusBadge } from '../components/StatusBadge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { TableCell, TableRow } from '@/components/ui/table'
import { useDomains } from '../hooks/useDomains'
import { delJson, getJson, postJson, putJson } from '../lib/api'
import { inboundSchema, type InboundFormInput, type InboundFormValues } from '../schemas/vless'
import type { InboundShare, ProxyInbound } from '../types'

const JsonView = lazy(() => import('@uiw/react-json-view'))

const defaults: InboundFormValues = {
  name: 'VLESS XHTTP Reality',
  template: 'vless-xhttp',
  domainId: 1,
  xhttpPath: '/xhttp',
  xhttpMode: 'auto',
  realityHandshakeServer: 'www.cloudflare.com',
  realityHandshakePort: 443,
  realityMaxTimeDiff: 60,
  enabled: true,
}

export function VlessPage() {
  const [items, setItems] = useState<ProxyInbound[]>([])
  const [editing, setEditing] = useState<ProxyInbound | null>(null)
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState<string>()
  const [configOpen, setConfigOpen] = useState(false)
  const [configDetails, setConfigDetails] = useState<Record<string, unknown> | null>(null)
  const [configTitle, setConfigTitle] = useState('')
  const [shareOpen, setShareOpen] = useState(false)
  const [share, setShare] = useState<InboundShare | null>(null)
  const [qrDataUrl, setQrDataUrl] = useState('')
  const { domains } = useDomains()

  const load = () => getJson<ProxyInbound[]>('inbounds').then(setItems)

  useEffect(() => {
    void load()
  }, [])

  const nextDefaults = useMemo<InboundFormValues>(() => ({
    ...defaults,
    domainId: domains[0]?.id ?? defaults.domainId,
  }), [domains])

  function openCreate() {
    setEditing(null)
    setOpen(true)
  }

  function openEdit(item: ProxyInbound) {
    setEditing(item)
    setOpen(true)
  }

  async function submit(values: InboundFormValues) {
    if (editing) {
      await putJson(`inbounds/${editing.id}`, values)
      toast.success('代理入口已更新')
    } else {
      await postJson('inbounds', values)
      toast.success('已新增代理入口')
    }
    setOpen(false)
    setEditing(null)
    void load()
  }

  async function action(item: ProxyInbound, name: string, run: () => Promise<unknown>) {
    const key = `${item.id}-${name}`
    setBusy(key)
    try {
      await run()
      toast.success('操作完成')
      void load()
    } catch {
      toast.error('操作失败')
    } finally {
      setBusy(undefined)
    }
  }

  async function showConfig(item: ProxyInbound) {
    const key = `${item.id}-config`
    setBusy(key)
    try {
      const result = await getJson<Record<string, unknown>>(`inbounds/${item.id}/config`)
      setConfigDetails(result)
      setConfigTitle(item.name)
      setConfigOpen(true)
    } catch {
      toast.error('读取配置详情失败')
    } finally {
      setBusy(undefined)
    }
  }

  async function fetchShare(item: ProxyInbound) {
    return getJson<InboundShare>(`inbounds/${item.id}/share`)
  }

  async function copyShare(item: ProxyInbound) {
    const key = `${item.id}-copy`
    setBusy(key)
    try {
      const result = await fetchShare(item)
      await navigator.clipboard.writeText(result.uri)
      toast.success('VLESS 链接已复制')
    } catch {
      toast.error('复制 VLESS 链接失败')
    } finally {
      setBusy(undefined)
    }
  }

  async function showShareQRCode(item: ProxyInbound) {
    const key = `${item.id}-qr`
    setBusy(key)
    try {
      const result = await fetchShare(item)
      const QRCode = await import('qrcode')
      const dataUrl = await QRCode.toDataURL(result.uri, { margin: 1, width: 256 })
      setShare(result)
      setQrDataUrl(dataUrl)
      setShareOpen(true)
    } catch {
      toast.error('生成二维码失败')
    } finally {
      setBusy(undefined)
    }
  }

  return (
    <div className="space-y-4" data-testid="inbounds-page">
      <div className="flex flex-wrap items-start justify-between gap-3" data-testid="inbounds-toolbar">
        <PageHeader title="代理入口" desc="管理 Xray VLESS XHTTP REALITY 公网入口。" data-testid="inbounds-header" />
        <Button onClick={openCreate} data-testid="inbound-create-button">
          <Plus size={16} aria-hidden="true" />
          新增入口
        </Button>
      </div>

      <DataTable headers={['名称', '客户端域名', '公网入口', 'XHTTP 路径', '状态', '操作']} data-testid="inbounds-table">
        {items.map(item => (
          <TableRow key={item.id} data-testid={`inbound-row-${item.id}`}>
            <TableCell>{item.name}</TableCell>
            <TableCell>{item.domain?.domain || domainNameForValue(domains, item.domainId, '-')}</TableCell>
            <TableCell>{formatInboundListen(item)}</TableCell>
            <TableCell>{item.xhttpPath || '/xhttp'}</TableCell>
            <TableCell><StatusBadge tone={item.enabled ? 'success' : 'neutral'}>{item.enabled ? '启用' : '停用'}</StatusBadge></TableCell>
            <TableCell>
              <div className="flex flex-wrap gap-2" data-testid={`inbound-actions-${item.id}`}>
                <Button variant="secondary" size="sm" onClick={() => openEdit(item)} data-testid={`inbound-edit-${item.id}`}>
                  <Edit size={15} aria-hidden="true" />
                  编辑
                </Button>
                <Button variant="secondary" size="sm" disabled={busy === `${item.id}-toggle`} onClick={() => action(item, 'toggle', () => postJson(`inbounds/${item.id}/${item.enabled ? 'disable' : 'enable'}`))} data-testid={`inbound-toggle-${item.id}`}>
                  {item.enabled ? <PowerOff size={15} aria-hidden="true" /> : <Power size={15} aria-hidden="true" />}
                  {item.enabled ? '停用' : '启用'}
                </Button>
                <Button variant="secondary" size="sm" disabled={busy === `${item.id}-config`} onClick={() => showConfig(item)} data-testid={`inbound-config-${item.id}`}>
                  <Code2 size={15} aria-hidden="true" />
                  配置详情
                </Button>
                <Button variant="secondary" size="sm" disabled={busy === `${item.id}-copy`} onClick={() => copyShare(item)} data-testid={`inbound-copy-link-${item.id}`}>
                  <Copy size={15} aria-hidden="true" />
                  复制链接
                </Button>
                <Button variant="secondary" size="sm" disabled={busy === `${item.id}-qr`} onClick={() => showShareQRCode(item)} data-testid={`inbound-show-qr-${item.id}`}>
                  <QrCode size={15} aria-hidden="true" />
                  二维码
                </Button>
                <Button variant="secondary" size="sm" disabled={busy === `${item.id}-delete`} onClick={() => action(item, 'delete', () => delJson(`inbounds/${item.id}`))} data-testid={`inbound-delete-${item.id}`}>
                  <Trash2 size={15} aria-hidden="true" />
                  删除
                </Button>
              </div>
            </TableCell>
          </TableRow>
        ))}
      </DataTable>

      {open ? (
        <InboundDialog
          domains={domains}
          initial={editing ? valuesFromItem(editing) : nextDefaults}
          title={editing ? '编辑代理入口' : '新增代理入口'}
          onClose={() => {
            setOpen(false)
            setEditing(null)
          }}
          onSubmit={submit}
        />
      ) : null}

      <Dialog open={configOpen} onOpenChange={setConfigOpen}>
        <DialogContent className="max-w-4xl" data-testid="inbound-config-dialog">
          <DialogHeader>
            <DialogTitle>{configTitle} 配置详情</DialogTitle>
          </DialogHeader>
          <div className="max-h-[70vh] overflow-auto rounded-md border border-neutral-200 bg-white p-3" data-testid="inbound-config-json">
            {configDetails ? (
              <Suspense fallback={<div className="text-sm text-muted-foreground" data-testid="inbound-config-json-loading">加载中…</div>}>
                <JsonView value={configDetails} collapsed={false} displayDataTypes={false} data-testid="inbound-config-json-view" />
              </Suspense>
            ) : null}
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={shareOpen} onOpenChange={setShareOpen}>
        <DialogContent className="max-w-md" data-testid="inbound-share-dialog">
          <DialogHeader>
            <DialogTitle>{share?.name ?? 'VLESS'} 分享二维码</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4" data-testid="inbound-share-content">
            {qrDataUrl ? (
              <img className="mx-auto size-64 rounded-md border border-neutral-200 bg-white p-2" src={qrDataUrl} alt="VLESS 链接二维码" data-testid="inbound-share-qr-image" />
            ) : null}
            <textarea
              className="min-h-24 resize-none rounded-md border border-neutral-200 bg-neutral-50 p-2 font-mono text-xs text-neutral-700"
              readOnly
              value={share?.uri ?? ''}
              data-testid="inbound-share-uri"
            />
            <DialogFooter>
              <Button variant="outline" type="button" onClick={() => share?.uri && navigator.clipboard.writeText(share.uri).then(() => toast.success('VLESS 链接已复制')).catch(() => toast.error('复制 VLESS 链接失败'))} data-testid="inbound-share-copy-button">
                <Copy size={15} aria-hidden="true" />
                复制链接
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function InboundDialog({
  domains,
  initial,
  title,
  onClose,
  onSubmit,
}: {
  domains: { id: number; domain: string }[]
  initial: InboundFormValues
  title: string
  onClose: () => void
  onSubmit: (values: InboundFormValues) => Promise<void>
}) {
  const { control, register, handleSubmit, formState: { errors, isSubmitting } } = useForm<InboundFormInput, unknown, InboundFormValues>({
    resolver: zodResolver(inboundSchema),
    defaultValues: initial,
  })

  return (
    <Dialog open onOpenChange={(nextOpen) => { if (!nextOpen) onClose() }}>
      <DialogContent className="max-w-3xl" data-testid="inbound-dialog">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <form className="grid gap-4" onSubmit={handleSubmit(onSubmit)} data-testid="inbound-form">
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="名称" error={errors.name?.message} data-testid="inbound-name-field">
              <Input {...register('name')} data-testid="inbound-name-input" />
            </FormField>
            <FormField
              label="客户端连接域名"
              description="客户端实际连接的域名，用于生成分享链接的 Host，并作为 XHTTP REALITY 的 serverName。"
              error={errors.domainId?.message}
              data-testid="inbound-domain-field"
            >
              <Controller
                control={control}
                name="domainId"
                render={({ field }) => (
                  <Select value={String(field.value)} onValueChange={(value) => field.onChange(Number(value))}>
                    <SelectTrigger className="w-full" data-testid="inbound-domain-select">
                      <SelectValue placeholder="选择域名…">
                        {value => domainNameForValue(domains, value, '选择域名…')}
                      </SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      {domains.map(domain => <SelectItem key={domain.id} value={String(domain.id)} label={domain.domain}>{domain.domain}</SelectItem>)}
                    </SelectContent>
                  </Select>
                )}
              />
            </FormField>
            <FormField
              label="公网入口"
              description="Xray 直接监听公网 HTTPS 端口 443；Nginx 不再转发 XHTTP 流量。"
              data-testid="inbound-public-entry-field"
            >
              <Input value="0.0.0.0:443" readOnly data-testid="inbound-public-entry-input" />
            </FormField>
            <FormField label="XHTTP 路径" error={errors.xhttpPath?.message} data-testid="inbound-xhttp-path-field">
              <Input {...register('xhttpPath')} data-testid="inbound-xhttp-path-input" />
            </FormField>
            <FormField label="XHTTP 模式" error={errors.xhttpMode?.message} data-testid="inbound-xhttp-mode-field">
              <Input {...register('xhttpMode')} data-testid="inbound-xhttp-mode-input" />
            </FormField>
            <FormField
              label="REALITY 握手服务器"
              description="REALITY 客户端使用的伪装 SNI，会写入 serverNames；普通 HTTPS 固定回落到内部 Nginx。"
              error={errors.realityHandshakeServer?.message}
              data-testid="inbound-handshake-server-field"
            >
              <Input {...register('realityHandshakeServer')} data-testid="inbound-handshake-server-input" />
            </FormField>
            <FormField
              label="REALITY 握手端口"
              description="REALITY 伪装握手服务器的端口，通常是 443。"
              error={errors.realityHandshakePort?.message}
              data-testid="inbound-handshake-port-field"
            >
              <Input type="number" {...register('realityHandshakePort')} data-testid="inbound-handshake-port-input" />
            </FormField>
            <FormField label="最大时间差" error={errors.realityMaxTimeDiff?.message} data-testid="inbound-max-time-diff-field">
              <Input type="number" {...register('realityMaxTimeDiff')} data-testid="inbound-max-time-diff-input" />
            </FormField>
          </div>

          <label className="inline-flex items-center gap-2 text-sm text-neutral-700" data-testid="inbound-enabled-field">
            <Controller
              control={control}
              name="enabled"
              render={({ field }) => (
                <Checkbox checked={field.value} onCheckedChange={field.onChange} data-testid="inbound-enabled-checkbox" />
              )}
            />
            启用入口
          </label>

          <DialogFooter>
            <Button variant="outline" type="button" onClick={onClose} data-testid="inbound-cancel-button">取消</Button>
            <Button type="submit" disabled={isSubmitting} data-testid="inbound-submit-button">{isSubmitting ? '保存中…' : '保存'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function valuesFromItem(item: ProxyInbound): InboundFormValues {
  return {
    name: item.name,
    template: 'vless-xhttp',
    domainId: item.domainId,
    xhttpPath: item.xhttpPath || '/xhttp',
    xhttpMode: item.xhttpMode || 'auto',
    realityHandshakeServer: item.realityHandshakeServer || 'www.cloudflare.com',
    realityHandshakePort: item.realityHandshakePort || 443,
    realityMaxTimeDiff: item.realityMaxTimeDiff || 60,
    enabled: item.enabled,
  }
}

function formatInboundListen(_item: ProxyInbound) {
  return '0.0.0.0:443'
}

function domainNameForValue(domains: { id: number; domain: string }[], value: unknown, fallback: string) {
  const id = Number(value)
  if (!Number.isFinite(id)) return fallback
  return domains.find(domain => domain.id === id)?.domain ?? fallback
}

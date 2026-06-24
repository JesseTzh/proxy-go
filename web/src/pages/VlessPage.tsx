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
  name: 'VLESS Reality Vision',
  template: 'vless-reality-vision',
  domainId: 1,
  listenPort: 31001,
  security: 'reality',
  xhttpPath: '/xhttp',
  xhttpMode: 'auto',
  realityHandshakeServer: 'www.cloudflare.com',
  realityHandshakePort: 443,
  realityMaxTimeDiff: 60,
  enabled: true,
}

const templateLabels: Record<ProxyInbound['template'], string> = {
  'vless-reality-vision': 'VLESS Reality Vision',
  'vless-xhttp': 'VLESS XHTTP',
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
        <PageHeader title="代理入口" desc="管理 Xray 入站模板与监听配置。" data-testid="inbounds-header" />
        <Button onClick={openCreate} data-testid="inbound-create-button">
          <Plus size={16} aria-hidden="true" />
          新增入口
        </Button>
      </div>

      <DataTable headers={['名称', '模板', '域名', '监听', '传输', '状态', '操作']} data-testid="inbounds-table">
        {items.map(item => (
          <TableRow key={item.id} data-testid={`inbound-row-${item.id}`}>
            <TableCell>{item.name}</TableCell>
            <TableCell>{templateLabels[item.template] ?? item.template}</TableCell>
            <TableCell>{item.domain?.domain || item.domainId}</TableCell>
            <TableCell>{item.listenAddr}:{item.listenPort}</TableCell>
            <TableCell>{item.network}{item.xhttpPath ? ` ${item.xhttpPath}` : ''}</TableCell>
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
  const { control, register, handleSubmit, watch, formState: { errors, isSubmitting } } = useForm<InboundFormInput, unknown, InboundFormValues>({
    resolver: zodResolver(inboundSchema),
    defaultValues: initial,
  })
  const template = watch('template')
  const showXHTTP = template === 'vless-xhttp'

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
            <FormField label="模板" error={errors.template?.message} data-testid="inbound-template-field">
              <Controller
                control={control}
                name="template"
                render={({ field }) => (
                  <Select value={field.value} onValueChange={field.onChange}>
                    <SelectTrigger className="w-full" data-testid="inbound-template-select">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="vless-reality-vision">VLESS Reality Vision</SelectItem>
                      <SelectItem value="vless-xhttp">VLESS XHTTP</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              />
            </FormField>
            <FormField label="域名" error={errors.domainId?.message} data-testid="inbound-domain-field">
              <Controller
                control={control}
                name="domainId"
                render={({ field }) => (
                  <Select value={String(field.value)} onValueChange={(value) => field.onChange(Number(value))}>
                    <SelectTrigger className="w-full" data-testid="inbound-domain-select">
                      <SelectValue placeholder="选择域名…" />
                    </SelectTrigger>
                    <SelectContent>
                      {domains.map(domain => <SelectItem key={domain.id} value={String(domain.id)}>{domain.domain}</SelectItem>)}
                    </SelectContent>
                  </Select>
                )}
              />
            </FormField>
            <FormField label="监听端口" error={errors.listenPort?.message} data-testid="inbound-listen-port-field">
              <Input type="number" {...register('listenPort')} data-testid="inbound-listen-port-input" />
            </FormField>
            <FormField label="安全层" error={errors.security?.message} data-testid="inbound-security-field">
              <Controller
                control={control}
                name="security"
                render={({ field }) => (
                  <Select value={field.value} onValueChange={field.onChange}>
                    <SelectTrigger className="w-full" data-testid="inbound-security-select">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="reality">REALITY</SelectItem>
                      <SelectItem value="tls">TLS</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              />
            </FormField>
            {showXHTTP ? (
              <>
                <FormField label="XHTTP 路径" error={errors.xhttpPath?.message} data-testid="inbound-xhttp-path-field">
                  <Input {...register('xhttpPath')} data-testid="inbound-xhttp-path-input" />
                </FormField>
                <FormField label="XHTTP 模式" error={errors.xhttpMode?.message} data-testid="inbound-xhttp-mode-field">
                  <Input {...register('xhttpMode')} data-testid="inbound-xhttp-mode-input" />
                </FormField>
              </>
            ) : null}
            <FormField label="握手服务器" error={errors.realityHandshakeServer?.message} data-testid="inbound-handshake-server-field">
              <Input {...register('realityHandshakeServer')} data-testid="inbound-handshake-server-input" />
            </FormField>
            <FormField label="握手端口" error={errors.realityHandshakePort?.message} data-testid="inbound-handshake-port-field">
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
    template: item.template,
    domainId: item.domainId,
    listenPort: item.listenPort,
    security: item.security === 'tls' ? 'tls' : 'reality',
    xhttpPath: item.xhttpPath || '/xhttp',
    xhttpMode: item.xhttpMode || 'auto',
    realityHandshakeServer: item.realityHandshakeServer || 'www.cloudflare.com',
    realityHandshakePort: item.realityHandshakePort || 443,
    realityMaxTimeDiff: item.realityMaxTimeDiff || 60,
    enabled: item.enabled,
  }
}

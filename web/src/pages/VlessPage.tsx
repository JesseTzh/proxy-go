import { useEffect, useMemo, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm } from 'react-hook-form'
import { Code2, Copy, Edit, Plus, Power, PowerOff, QrCode, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { DataTable } from '../components/DataTable'
import { FormField } from '../components/FormField'
import { JsonPreview } from '../components/JsonPreview'
import { PageHeader } from '../components/PageHeader'
import { StatusBadge } from '../components/StatusBadge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { TableCell, TableRow } from '@/components/ui/table'
import { useDomains } from '../hooks/useDomains'
import { delJson, getJson, postJson, putJson } from '../lib/api'
import { inboundSchema, type InboundFormInput, type InboundFormValues } from '../schemas/vless'
import type { InboundShare, ProxyInbound } from '../types'

const defaults: InboundFormValues = {
  template: 'vless-reality-vision',
  name: 'VLESS Reality Vision',
  domainId: 1,
  realityHandshakeServer: 'apple.com',
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
      // global error dialog handles API failures
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
      // global error dialog handles API failures
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
      toast.success('代理链接已复制')
    } catch {
      toast.error('复制代理链接失败')
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
        <PageHeader title="代理入口" desc="管理经 Nginx SNI 分流进入 sing-box 的 Reality Vision 与 AnyTLS 入站。" data-testid="inbounds-header" />
        <Button onClick={openCreate} data-testid="inbound-create-button">
          <Plus size={16} aria-hidden="true" />
          新增入口
        </Button>
      </div>

      <DataTable headers={['名称', '协议', '客户端域名', '分流 SNI', '本地监听', '状态', '操作']} data-testid="inbounds-table">
        {items.map(item => (
          <TableRow key={item.id} data-testid={`inbound-row-${item.id}`}>
            <TableCell>{item.name}</TableCell>
            <TableCell>{protocolLabel(item.template)}</TableCell>
            <TableCell>{item.domain?.domain || domainNameForValue(domains, item.domainId, '-')}</TableCell>
            <TableCell>{item.routeSni || '-'}</TableCell>
            <TableCell>{formatInboundListen(item)}</TableCell>
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
                <Button variant="secondary" size="sm" disabled={busy === `${item.id}-copy`} onClick={() => copyShare(item)}>
                  <Copy size={15} aria-hidden="true" />
                  复制链接
                </Button>
                <Button variant="secondary" size="sm" disabled={busy === `${item.id}-qr`} onClick={() => showShareQRCode(item)} data-testid={`inbound-show-qr-${item.id}`}>
                  <QrCode size={15} aria-hidden="true" />
                  获取链接/二维码
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
          {configDetails ? <JsonPreview value={configDetails} data-testid="inbound-config-json" /> : null}
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
              <Button variant="outline" type="button" onClick={() => share?.uri && navigator.clipboard.writeText(share.uri).then(() => toast.success('代理链接已复制')).catch(() => toast.error('复制代理链接失败'))} data-testid="inbound-share-copy-button">
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
  const { control, register, watch, handleSubmit, formState: { errors, isSubmitting } } = useForm<InboundFormInput, unknown, InboundFormValues>({
    resolver: zodResolver(inboundSchema),
    defaultValues: initial,
  })
  const template = watch('template')

  return (
    <Dialog open onOpenChange={(nextOpen) => { if (!nextOpen) onClose() }}>
      <DialogContent className="max-w-3xl" data-testid="inbound-dialog">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <form className="grid gap-5" onSubmit={handleSubmit(onSubmit)} data-testid="inbound-form">
          <div
            className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900"
            data-testid="inbound-public-entry-note"
          >
            公网 443 由 Nginx stream 统一监听；匹配分流 SNI 后透传到本机 sing-box 入站。
          </div>

          <section className="grid gap-3" data-testid="inbound-basic-section">
            <h3 className="text-sm font-medium text-neutral-900" data-testid="inbound-basic-section-title">基础信息</h3>
            <div className="grid gap-4 md:grid-cols-2" data-testid="inbound-basic-section-fields">
              <FormField label="协议模板" error={errors.template?.message} data-testid="inbound-template-field">
                <Controller
                  control={control}
                  name="template"
                  render={({ field }) => (
                    <Select value={field.value} onValueChange={field.onChange}>
                      <SelectTrigger className="w-full" data-testid="inbound-template-select">
                        <SelectValue placeholder="选择协议…" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="vless-reality-vision" label="VLESS Reality Vision">VLESS Reality Vision</SelectItem>
                        <SelectItem value="anytls" label="AnyTLS">AnyTLS</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                />
              </FormField>
              <FormField label="名称" error={errors.name?.message} data-testid="inbound-name-field">
                <Input {...register('name')} data-testid="inbound-name-input" />
              </FormField>
            </div>
          </section>

          <section className="grid gap-3" data-testid="inbound-entry-section">
            <h3 className="text-sm font-medium text-neutral-900" data-testid="inbound-entry-section-title">连接入口</h3>
            <div className="grid gap-4 md:grid-cols-2" data-testid="inbound-entry-section-fields">
              <FormField
                label="客户端连接域名"
                description={template === 'anytls' ? 'AnyTLS 使用该域名作为 TLS 证书域名和 SNI 分流入口。' : 'Reality Vision 客户端实际连接的域名，用于生成分享链接地址。'}
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
            </div>
          </section>

          {template === 'vless-reality-vision' ? (
            <section className="grid gap-3" data-testid="inbound-reality-section">
              <h3 className="text-sm font-medium text-neutral-900" data-testid="inbound-reality-section-title">REALITY</h3>
              <div className="grid gap-4 md:grid-cols-2" data-testid="inbound-reality-section-fields">
                <FormField
                  label="REALITY 握手服务器"
                  description="Reality Vision 客户端使用的伪装 SNI，会写入分享链接 sni、Nginx stream 分流规则和 sing-box reality handshake，例如 apple.com。不要填写已托管域名。"
                  error={errors.realityHandshakeServer?.message}
                  data-testid="inbound-handshake-server-field"
                >
                  <Input {...register('realityHandshakeServer')} data-testid="inbound-handshake-server-input" />
                </FormField>
              </div>
            </section>
          ) : null}

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
    template: item.template,
    name: item.name,
    domainId: item.domainId,
    realityHandshakeServer: item.realityHandshakeServer || 'apple.com',
  }
}

function formatInboundListen(item: ProxyInbound) {
  const local = item.listenAddr && item.listenPort ? `${item.listenAddr}:${item.listenPort}` : '-'
  return `0.0.0.0:443 -> ${local}`
}

function protocolLabel(template: ProxyInbound['template']) {
  return template === 'anytls' ? 'AnyTLS' : 'VLESS Reality Vision'
}

function domainNameForValue(domains: { id: number; domain: string }[], value: unknown, fallback: string) {
  const id = Number(value)
  if (!Number.isFinite(id)) return fallback
  return domains.find(domain => domain.id === id)?.domain ?? fallback
}

import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm } from 'react-hook-form'
import { Pencil, Save, Trash2, X } from 'lucide-react'
import { toast } from 'sonner'
import { DataTable } from '../components/DataTable'
import { FormField } from '../components/FormField'
import { PageHeader } from '../components/PageHeader'
import { StatusBadge } from '../components/StatusBadge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { TableCell, TableRow } from '@/components/ui/table'
import { useDomains } from '../hooks/useDomains'
import { delJson, getJson, postJson, putJson } from '../lib/api'
import { reverseProxySchema, type ReverseProxyFormInput, type ReverseProxyFormValues } from '../schemas/reverseProxy'
import type { ReverseProxy } from '../types'

const LOCAL_HOST_TARGET = 'host.docker.internal'

export function ReverseProxiesPage(){
  const [items,setItems]=useState<ReverseProxy[]>([])
  const [useLocalPort,setUseLocalPort]=useState(false)
  const [editingId,setEditingId]=useState<number>()
  const [editValues,setEditValues]=useState<ReverseProxyFormInput>()
  const {domains,loading: domainsLoading}=useDomains()
  const {control,register,handleSubmit,setValue,watch,formState:{errors}} = useForm<ReverseProxyFormInput, unknown, ReverseProxyFormValues>({
    resolver:zodResolver(reverseProxySchema),
    defaultValues:{
      domainId:0,
      targetScheme:'http',
      targetHost:'127.0.0.1',
      targetPort:8080,
      preserveHost:true,
      webSocket:true,
      passRealIp:true,
      enabled:true,
      remark:'',
    },
  })
  const selectedDomainId = watch('domainId')
  const load=()=>getJson<ReverseProxy[]>('reverse-proxies').then(setItems)
  useEffect(()=>{ void load() },[])
  useEffect(()=>{
    if (domains.length === 0) return
    if (!domains.some(domain => domain.id === selectedDomainId)) {
      setValue('domainId', domains[0].id, { shouldValidate: true })
    }
  },[domains, selectedDomainId, setValue])
  async function add(values: ReverseProxyFormValues){
    await postJson('reverse-proxies',values)
    toast.success('已新增反代规则，配置已自动应用')
    load()
  }

  function startEdit(item: ReverseProxy) {
    setEditingId(item.id)
    setEditValues({
      domainId: item.domainId,
      targetScheme: item.targetScheme === 'https' ? 'https' : 'http',
      targetHost: item.targetHost,
      targetPort: item.targetPort,
      preserveHost: item.preserveHost,
      webSocket: item.webSocket,
      passRealIp: item.passRealIp,
      enabled: item.enabled,
      remark: item.remark ?? '',
    })
  }

  function updateEditValue<K extends keyof ReverseProxyFormInput>(key: K, value: ReverseProxyFormInput[K]) {
    setEditValues(current => current ? { ...current, [key]: value } : current)
  }

  async function saveEdit(id: number) {
    const parsed = reverseProxySchema.safeParse(editValues)
    if (!parsed.success) {
      toast.error(parsed.error.issues[0]?.message ?? '请检查反代规则')
      return
    }
    await putJson(`reverse-proxies/${id}`, parsed.data)
    toast.success('反代规则已更新，配置已自动应用')
    setEditingId(undefined)
    setEditValues(undefined)
    load()
  }

  async function remove(id: number) {
    await delJson(`reverse-proxies/${id}`)
    toast.success('反代规则已删除，配置已自动应用')
    load()
  }

  function toggleLocalPort(checked: boolean) {
    setUseLocalPort(checked)
    if (checked) {
      setValue('targetHost', LOCAL_HOST_TARGET, { shouldDirty: true, shouldValidate: true })
    }
  }

  return (
    <div className="space-y-4" data-testid="reverse-proxies-page">
      <PageHeader title="反向代理" desc="支持本机/内网 HTTP 与 HTTPS 目标。" data-testid="reverse-proxies-header"/>
      <Card className="p-4">
        <form onSubmit={handleSubmit(add)} className="grid gap-3 md:grid-cols-5" data-testid="reverse-proxy-create-form">
        <FormField label="域名" error={errors.domainId?.message} data-testid="reverse-domain-field">
          <Controller
            control={control}
            name="domainId"
            render={({ field }) => (
              <Select value={field.value ? String(field.value) : undefined} onValueChange={value => field.onChange(Number(value))} disabled={domainsLoading || domains.length === 0}>
                <SelectTrigger className="w-full" data-testid="reverse-domain-select">
                  <SelectValue placeholder={domainsLoading ? '加载域名…' : '选择域名…'}>
                    {value => domainNameForValue(domains, value, domainsLoading ? '加载域名…' : '选择域名…')}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>{domains.map(d=><SelectItem key={d.id} value={String(d.id)} label={d.domain}>{d.domain}</SelectItem>)}</SelectContent>
              </Select>
            )}
          />
        </FormField>
        <FormField label="协议" error={errors.targetScheme?.message} data-testid="reverse-scheme-field">
          <Controller
            control={control}
            name="targetScheme"
            render={({ field }) => (
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger className="w-full" data-testid="reverse-scheme-select"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="http">http</SelectItem>
                  <SelectItem value="https">https</SelectItem>
                </SelectContent>
              </Select>
            )}
          />
        </FormField>
        <FormField label="目标主机" error={errors.targetHost?.message} data-testid="reverse-target-host-field">
          <Input readOnly={useLocalPort} {...register('targetHost')} data-testid="reverse-target-host-input"/>
        </FormField>
        <FormField label="目标端口" error={errors.targetPort?.message} data-testid="reverse-target-port-field">
          <Input type="number" {...register('targetPort')} data-testid="reverse-target-port-input"/>
        </FormField>
        <Button type="submit" className="self-end" disabled={domains.length === 0} data-testid="reverse-create-button">新增</Button>
        <Toggle label="代理本地端口" checked={useLocalPort} onChange={toggleLocalPort} data-testid="reverse-local-port-toggle" />
        <FormCheckbox control={control} name="preserveHost" label="Preserve Host" data-testid="reverse-preserve-host-toggle" />
        <FormCheckbox control={control} name="webSocket" label="WebSocket" data-testid="reverse-websocket-toggle" />
        <FormCheckbox control={control} name="passRealIp" label="Real IP" data-testid="reverse-real-ip-toggle" />
        <FormCheckbox control={control} name="enabled" label="启用" data-testid="reverse-enabled-toggle" />
        <input type="hidden" {...register('remark')}/>
        </form>
      </Card>
      <DataTable headers={['域名','目标','WebSocket','真实IP','状态','操作']} data-testid="reverse-proxies-table">
        {items.map(x=>{
          const editing = editingId === x.id && editValues
          return (
          <TableRow key={x.id} data-testid={`reverse-row-${x.id}`}>
            <TableCell data-testid={`reverse-domain-${x.id}`}>
              {editing ? (
                <Select value={String(editValues.domainId || '')} onValueChange={value => updateEditValue('domainId', Number(value))} disabled={domainsLoading || domains.length === 0}>
                  <SelectTrigger className="w-44" data-testid={`reverse-edit-domain-${x.id}`}>
                    <SelectValue>{value => domainNameForValue(domains, value, '选择域名…')}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>{domains.map(d=><SelectItem key={d.id} value={String(d.id)} label={d.domain}>{d.domain}</SelectItem>)}</SelectContent>
                </Select>
              ) : x.domain?.domain || domainNameForValue(domains, x.domainId, '-')}
            </TableCell>
            <TableCell data-testid={`reverse-target-${x.id}`}>
              {editing ? (
                <div className="grid min-w-80 grid-cols-[96px_minmax(120px,1fr)_88px] gap-2" data-testid={`reverse-edit-target-${x.id}`}>
                  <Select value={String(editValues.targetScheme)} onValueChange={value => updateEditValue('targetScheme', value === 'https' ? 'https' : 'http')}>
                    <SelectTrigger data-testid={`reverse-edit-scheme-${x.id}`}><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="http">http</SelectItem>
                      <SelectItem value="https">https</SelectItem>
                    </SelectContent>
                  </Select>
                  <Input value={String(editValues.targetHost ?? '')} onChange={event => updateEditValue('targetHost', event.target.value)} data-testid={`reverse-edit-host-${x.id}`}/>
                  <Input type="number" value={String(editValues.targetPort ?? '')} onChange={event => updateEditValue('targetPort', event.target.value)} data-testid={`reverse-edit-port-${x.id}`}/>
                </div>
              ) : `${x.targetScheme}://${x.targetHost}:${x.targetPort}`}
            </TableCell>
            <TableCell data-testid={`reverse-websocket-${x.id}`}>
              {editing ? (
                <InlineCheckbox checked={Boolean(editValues.webSocket)} onChange={value => updateEditValue('webSocket', value)} data-testid={`reverse-edit-websocket-${x.id}`} />
              ) : x.webSocket?'是':'否'}
            </TableCell>
            <TableCell data-testid={`reverse-real-ip-${x.id}`}>
              {editing ? (
                <InlineCheckbox checked={Boolean(editValues.passRealIp)} onChange={value => updateEditValue('passRealIp', value)} data-testid={`reverse-edit-real-ip-${x.id}`} />
              ) : x.passRealIp?'是':'否'}
            </TableCell>
            <TableCell data-testid={`reverse-status-${x.id}`}>
              {editing ? (
                <InlineCheckbox checked={Boolean(editValues.enabled)} onChange={value => updateEditValue('enabled', value)} data-testid={`reverse-edit-enabled-${x.id}`} />
              ) : <StatusBadge tone={x.enabled ? 'success' : 'neutral'}>{x.enabled?'启用':'禁用'}</StatusBadge>}
            </TableCell>
            <TableCell data-testid={`reverse-actions-${x.id}`}>
              {editing ? (
                <div className="flex items-center gap-2" data-testid={`reverse-edit-actions-${x.id}`}>
                  <Button variant="outline" size="sm" onClick={()=>void saveEdit(x.id)} data-testid={`reverse-save-${x.id}`}>
                    <Save size={14} aria-hidden="true" />
                    保存
                  </Button>
                  <Button variant="ghost" size="sm" onClick={()=>{ setEditingId(undefined); setEditValues(undefined) }} data-testid={`reverse-cancel-${x.id}`}>
                    <X size={14} aria-hidden="true" />
                    取消
                  </Button>
                </div>
              ) : (
                <div className="flex items-center gap-2" data-testid={`reverse-view-actions-${x.id}`}>
                  <Button variant="outline" size="sm" onClick={()=>startEdit(x)} data-testid={`reverse-edit-${x.id}`}>
                    <Pencil size={14} aria-hidden="true" />
                    编辑
                  </Button>
                  <Button variant="outline" size="sm" onClick={()=>void remove(x.id)} data-testid={`reverse-delete-${x.id}`}>
                    <Trash2 size={14} aria-hidden="true" />
                    删除
                  </Button>
                </div>
              )}
            </TableCell>
          </TableRow>
        )})}
      </DataTable>
    </div>
  )
}

function InlineCheckbox({ checked, onChange, 'data-testid': dataTestId }: { checked: boolean; onChange: (checked: boolean) => void; 'data-testid'?: string }) {
  return (
    <Checkbox checked={checked} onCheckedChange={value => onChange(Boolean(value))} data-testid={dataTestId} />
  )
}

function Toggle({ label, checked, onChange, 'data-testid': dataTestId }: { label: string; checked: boolean; onChange: (checked: boolean) => void; 'data-testid'?: string }) {
  return (
    <label className="inline-flex items-center gap-2 text-sm text-neutral-700" data-testid={dataTestId}>
      <Checkbox checked={checked} onCheckedChange={value => onChange(Boolean(value))} />
      {label}
    </label>
  )
}

function FormCheckbox({ control, name, label, 'data-testid': dataTestId }: { control: ReturnType<typeof useForm<ReverseProxyFormInput, unknown, ReverseProxyFormValues>>['control']; name: 'preserveHost' | 'webSocket' | 'passRealIp' | 'enabled'; label: string; 'data-testid'?: string }) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field }) => (
        <label className="inline-flex items-center gap-2 text-sm text-neutral-700" data-testid={dataTestId}>
          <Checkbox checked={field.value} onCheckedChange={field.onChange} />
          {label}
        </label>
      )}
    />
  )
}

function domainNameForValue(domains: { id: number; domain: string }[], value: unknown, fallback: string) {
  const id = Number(value)
  if (!Number.isFinite(id)) return fallback
  return domains.find(domain => domain.id === id)?.domain ?? fallback
}

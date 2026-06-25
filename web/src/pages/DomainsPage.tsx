import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { BadgeCheck, FileKey2, RefreshCw, Search, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { DataTable } from '../components/DataTable'
import { FormField } from '../components/FormField'
import { PageHeader } from '../components/PageHeader'
import { StatusBadge } from '../components/StatusBadge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { TableCell, TableRow } from '@/components/ui/table'
import { delJson, getJson, postJson } from '../lib/api'
import { domainSchema, type DomainFormValues } from '../schemas/domain'
import type { Domain } from '../types'

export function DomainsPage() {
  const [items, setItems] = useState<Domain[]>([])
  const [busy, setBusy] = useState<string>()
  const { register, handleSubmit, reset, formState: { errors } } = useForm<DomainFormValues>({
    resolver: zodResolver(domainSchema),
    defaultValues: { domain: '' },
  })

  const load = () => getJson<Domain[]>('domains').then(setItems)

  useEffect(() => {
    void load()
  }, [])

  async function add(values: DomainFormValues) {
    await postJson('domains', { ...values, remark: '', status: 'enabled' })
    reset()
    toast.success('已新增域名')
    void load()
  }

  async function domainAction(domain: Domain, action: 'dns' | 'issue' | 'renew' | 'delete-cert' | 'delete-domain') {
    const key = `${domain.id}-${action}`
    setBusy(key)
    try {
      if (action === 'dns') {
        const result = await postJson(`domains/${domain.id}/dns-check`)
        toast(JSON.stringify(result))
      }
      if (action === 'issue') {
        await postJson(`domains/${domain.id}/certificate/issue`)
        toast.success('已触发证书申请')
      }
      if (action === 'renew') {
        await postJson(`domains/${domain.id}/certificate/renew`)
        toast.success('已触发证书续期')
      }
      if (action === 'delete-cert') {
        await delJson(`domains/${domain.id}/certificate`)
        toast.success('已删除证书')
      }
      if (action === 'delete-domain') {
        await delJson(`domains/${domain.id}`)
        toast.success('已删除域名')
      }
      void load()
    } catch {
      // global error dialog handles API failures
    } finally {
      setBusy(undefined)
    }
  }

  return (
    <div className="space-y-4" data-testid="domains-page">
      <PageHeader title="域名管理" desc="域名、DNS 检查与证书申请集中在同一资源下管理。" data-testid="domains-header" />

      <Card className="p-4">
        <form onSubmit={handleSubmit(add)} className="grid gap-3 md:grid-cols-[1fr_auto] md:items-end" data-testid="domain-create-form">
        <FormField label="域名" error={errors.domain?.message} data-testid="domain-create-field">
          <Input placeholder="example.com…" {...register('domain')} data-testid="domain-create-input" />
        </FormField>
        <Button type="submit" data-testid="domain-create-button">新增域名</Button>
      </form>
      </Card>

      <DataTable headers={['域名', '状态', '证书', '到期时间', '错误', '操作']} data-testid="domains-table">
        {items.map(item => {
          const cert = item.certificate
          return (
            <TableRow key={item.id} data-testid={`domain-row-${item.id}`}>
              <TableCell>
                <div className="font-medium">{item.domain}</div>
                {item.remark ? <div className="text-xs text-slate-500 mt-1">{item.remark}</div> : null}
              </TableCell>
              <TableCell><StatusBadge tone={item.status === 'enabled' ? 'success' : 'neutral'}>{item.status}</StatusBadge></TableCell>
              <TableCell>
                {cert ? (
                  <span className="inline-flex items-center gap-1 text-sm">
                    <BadgeCheck size={14} />
                    {cert.status || 'unknown'}
                  </span>
                ) : (
                  <span className="text-slate-500">未申请</span>
                )}
              </TableCell>
              <TableCell>{cert?.expiresAt || '-'}</TableCell>
              <TableCell className="max-w-72 break-words">{cert?.errorMessage || '-'}</TableCell>
              <TableCell>
                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" size="sm" disabled={busy === `${item.id}-dns`} onClick={() => domainAction(item, 'dns')} data-testid={`domain-dns-${item.id}`}>
                    <Search size={15} aria-hidden="true" />
                    DNS
                  </Button>
                  <Button variant="outline" size="sm" disabled={busy === `${item.id}-issue`} onClick={() => domainAction(item, 'issue')} data-testid={`domain-issue-${item.id}`}>
                    <FileKey2 size={15} aria-hidden="true" />
                    {cert ? '重新申请' : '申请证书'}
                  </Button>
                  {cert ? (
                    <>
                      <Button variant="outline" size="sm" disabled={busy === `${item.id}-renew`} onClick={() => domainAction(item, 'renew')} data-testid={`domain-renew-${item.id}`}>
                        <RefreshCw size={15} aria-hidden="true" />
                        续期
                      </Button>
                      <Button variant="outline" size="sm" disabled={busy === `${item.id}-delete-cert`} onClick={() => domainAction(item, 'delete-cert')} data-testid={`domain-delete-cert-${item.id}`}>
                        <Trash2 size={15} aria-hidden="true" />
                        删除证书
                      </Button>
                    </>
                  ) : null}
                  <Button variant="outline" size="sm" disabled={busy === `${item.id}-delete-domain`} onClick={() => domainAction(item, 'delete-domain')} data-testid={`domain-delete-${item.id}`}>
                    <Trash2 size={15} aria-hidden="true" />
                    删除域名
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          )
        })}
      </DataTable>
    </div>
  )
}

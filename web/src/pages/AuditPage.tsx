import { useEffect, useState } from 'react'
import { Eye } from 'lucide-react'
import { DataTable } from '../components/DataTable'
import { JsonPreview } from '../components/JsonPreview'
import { PageHeader } from '../components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { TableCell, TableRow } from '@/components/ui/table'
import { getJson } from '../lib/api'
import type { AuditLog } from '../types'

export function AuditPage() {
  const [items, setItems] = useState<AuditLog[]>([])
  const [detailItem, setDetailItem] = useState<AuditLog | null>(null)

  useEffect(() => {
    getJson<AuditLog[]>('audit-logs').then(setItems)
  }, [])

  return (
    <div className="space-y-4" data-testid="audit-page">
      <PageHeader title="审计日志" desc="登录、证书、域名、反代、代理入口、Runtime 操作记录。" data-testid="audit-header" />
      <DataTable headers={['时间', '操作', '资源', 'IP', 'User-Agent', '操作']} data-testid="audit-table">
        {items.map(x => (
          <TableRow key={x.id} data-testid={`audit-row-${x.id}`}>
            <TableCell data-testid={`audit-created-at-${x.id}`}>{formatAuditTime(x.createdAt)}</TableCell>
            <TableCell data-testid={`audit-action-${x.id}`}>{x.action}</TableCell>
            <TableCell data-testid={`audit-resource-${x.id}`}>{x.resourceType}:{x.resourceId}</TableCell>
            <TableCell data-testid={`audit-ip-${x.id}`}>{x.ip}</TableCell>
            <TableCell data-testid={`audit-user-agent-${x.id}`}>{x.userAgent}</TableCell>
            <TableCell>
              <Button variant="secondary" size="sm" onClick={() => setDetailItem(x)} data-testid={`audit-detail-${x.id}`}>
                <Eye size={15} aria-hidden="true" />
                详情
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </DataTable>

      <Dialog open={Boolean(detailItem)} onOpenChange={(nextOpen) => { if (!nextOpen) setDetailItem(null) }}>
        <DialogContent className="max-w-4xl" data-testid="audit-detail-dialog">
          <DialogHeader>
            <DialogTitle>审计详情</DialogTitle>
          </DialogHeader>
          <JsonPreview value={parseAuditDetail(detailItem?.detail)} data-testid="audit-detail-json" />
        </DialogContent>
      </Dialog>
    </div>
  )
}

function formatAuditTime(value: string) {
  return value.replace(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.\d+)?(Z|[+-]\d{2}:\d{2})$/, '$1 $2')
}

function parseAuditDetail(value?: string) {
  if (!value) return {}
  try {
    return JSON.parse(value)
  } catch {
    return { detail: value }
  }
}

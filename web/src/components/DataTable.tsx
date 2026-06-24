import type { ReactNode } from 'react'
import {
  Table,
  TableBody,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

export function DataTable({
  headers,
  children,
  'data-testid': dataTestId,
}: {
  headers: string[]
  children: ReactNode
  'data-testid'?: string
}) {
  return (
    <div className="rounded-xl bg-card shadow-[var(--shadow-border)]" data-testid={dataTestId}>
      <Table>
        <TableHeader>
          <TableRow>{headers.map(h => <TableHead key={h}>{h}</TableHead>)}</TableRow>
        </TableHeader>
        <TableBody>{children}</TableBody>
      </Table>
    </div>
  )
}

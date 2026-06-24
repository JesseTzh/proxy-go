import type { ReactNode } from 'react'
import { Badge } from '@/components/ui/badge'

export function StatusBadge({ children, tone = 'neutral' }: { children: ReactNode; tone?: 'neutral' | 'success' | 'warning' | 'danger' }) {
  const classes = {
    neutral: 'bg-secondary text-secondary-foreground',
    success: 'bg-secondary text-secondary-foreground',
    warning: 'bg-[#de1d8d]/10 text-[#171717]',
    danger: 'bg-[#ff5b4f]/10 text-[#171717]',
  }[tone]

  return <Badge variant="secondary" className={classes}>{children}</Badge>
}

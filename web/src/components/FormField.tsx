import type { ReactNode } from 'react'
import { HelpCircle } from 'lucide-react'

export function FormField({
  label,
  description,
  error,
  children,
  'data-testid': dataTestId,
}: {
  label: string
  description?: string
  error?: string
  children: ReactNode
  'data-testid'?: string
}) {
  return (
    <label className="grid gap-1.5 text-sm" data-testid={dataTestId}>
      <span className="flex items-center gap-1.5 text-neutral-600">
        {label}
        {description ? (
          <span className="group relative inline-flex" data-testid={dataTestId ? `${dataTestId}-help` : undefined}>
            <HelpCircle
              size={14}
              aria-label={description}
              className="text-muted-foreground"
              data-testid={dataTestId ? `${dataTestId}-help-icon` : undefined}
              tabIndex={0}
            />
            <span
              role="tooltip"
              className="pointer-events-none absolute left-1/2 top-5 z-50 hidden w-72 -translate-x-1/2 rounded-md border border-neutral-200 bg-popover px-3 py-2 text-xs leading-5 text-popover-foreground shadow-md group-hover:block group-focus-within:block"
              data-testid={dataTestId ? `${dataTestId}-tooltip` : undefined}
            >
              {description}
            </span>
          </span>
        ) : null}
      </span>
      {children}
      {error ? <span className="text-sm text-red-600" aria-live="polite">{error}</span> : null}
    </label>
  )
}

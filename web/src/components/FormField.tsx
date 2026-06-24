import type { ReactNode } from 'react'

export function FormField({
  label,
  error,
  children,
  'data-testid': dataTestId,
}: {
  label: string
  error?: string
  children: ReactNode
  'data-testid'?: string
}) {
  return (
    <label className="grid gap-1.5 text-sm" data-testid={dataTestId}>
      <span className="text-neutral-600">{label}</span>
      {children}
      {error ? <span className="text-sm text-red-600" aria-live="polite">{error}</span> : null}
    </label>
  )
}

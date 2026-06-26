import { lazy, Suspense } from 'react'

const JsonView = lazy(() => import('@uiw/react-json-view'))

export function JsonPreview({
  value,
  'data-testid': dataTestId,
}: {
  value: unknown
  'data-testid'?: string
}) {
  const jsonValue = normalizeJsonValue(value)

  return (
    <div className="max-h-[70vh] overflow-auto rounded-md border border-neutral-200 bg-white p-3" data-testid={dataTestId}>
      <Suspense fallback={<div className="text-sm text-muted-foreground" data-testid={dataTestId ? `${dataTestId}-loading` : undefined}>加载中...</div>}>
        <JsonView value={jsonValue} collapsed={false} displayDataTypes={false} data-testid={dataTestId ? `${dataTestId}-view` : undefined} />
      </Suspense>
    </div>
  )
}

function normalizeJsonValue(value: unknown): object {
  if (value !== null && typeof value === 'object') return value
  return { value }
}

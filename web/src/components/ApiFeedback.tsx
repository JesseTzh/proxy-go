import { Loader2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { clearApiError, subscribeApiUi } from '../lib/apiUi'

export function ApiFeedback() {
  const [pendingCount, setPendingCount] = useState(0)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const unsubscribe = subscribeApiUi(state => {
      setPendingCount(state.pendingCount)
      setError(state.error)
    })
    return () => {
      unsubscribe()
    }
  }, [])

  return (
    <>
      {pendingCount > 0 ? (
        <div className="fixed inset-0 z-[60] grid place-items-center bg-white/65 backdrop-blur-sm" data-testid="api-loading-overlay">
          <div className="flex min-w-56 items-center gap-3 rounded-lg border border-neutral-200 bg-white px-4 py-3 text-sm font-medium text-neutral-800 shadow-lg" data-testid="api-loading-panel">
            <Loader2 className="size-5 animate-spin text-neutral-600" aria-hidden="true" />
            <span data-testid="api-loading-text">处理中，请稍候…</span>
          </div>
        </div>
      ) : null}
      <Dialog open={Boolean(error)} onOpenChange={open => { if (!open) clearApiError() }}>
        <DialogContent data-testid="api-error-dialog">
          <DialogHeader data-testid="api-error-header">
            <DialogTitle data-testid="api-error-title">操作失败</DialogTitle>
            <DialogDescription className="break-words whitespace-pre-wrap" data-testid="api-error-message">
              {error}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter data-testid="api-error-footer">
            <Button type="button" onClick={clearApiError} data-testid="api-error-confirm-button">知道了</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

import { useEffect, useState } from 'react'
import { Bug, LogIn, LogOut, RotateCcw, Trash2, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { clearMockRequests, getMockState, isMockMode, resetMockState, setMockAuthenticated, setMockDelay, setMockFailNext, subscribeMock, type MockRequest } from '../lib/mockApi'

export function MockDebugPanel() {
  const [open, setOpen] = useState(false)
  const [state, setState] = useState(getMockState())
  useEffect(() => subscribeMock(() => setState({ ...getMockState(), requests: [...getMockState().requests] })), [])
  if (!isMockMode) return null
  return (
    <aside className="fixed bottom-4 right-4 z-[70] w-[min(420px,calc(100vw-2rem))]" data-testid="mock-debug-panel">
      {!open ? <Button onClick={() => setOpen(true)} className="shadow-lg" data-testid="mock-debug-open"><Bug size={16} /> Mock 调试</Button> : (
        <Card className="overflow-hidden border-neutral-300 bg-white shadow-2xl" data-testid="mock-debug-card">
          <div className="flex items-center justify-between border-b border-neutral-200 bg-neutral-950 px-4 py-3 text-white" data-testid="mock-debug-header">
            <div className="flex items-center gap-2 text-sm font-semibold"><Bug size={16} /> Mock API 调试</div>
            <Button variant="ghost" size="icon" className="text-white hover:bg-white/10 hover:text-white" onClick={() => setOpen(false)} data-testid="mock-debug-close" aria-label="关闭 Mock 调试"><X size={16} /></Button>
          </div>
          <div className="grid gap-4 p-4 text-sm" data-testid="mock-debug-controls">
            <label className="grid gap-1" data-testid="mock-debug-delay-field">响应延迟（毫秒）<Input type="number" min={0} max={3000} value={state.delay} onChange={event => setMockDelay(Number(event.target.value))} data-testid="mock-debug-delay-input" /></label>
            <div className="flex items-center justify-between" data-testid="mock-debug-auth-field"><span>模拟登录状态：{state.authenticated ? '已登录' : '未登录'}</span><Button variant="outline" size="sm" onClick={() => setMockAuthenticated(!state.authenticated)} data-testid="mock-debug-auth-toggle">{state.authenticated ? <LogOut size={14} /> : <LogIn size={14} />}{state.authenticated ? '退出' : '登录'}</Button></div>
            <label className="flex items-center gap-2" data-testid="mock-debug-fail-field"><Checkbox checked={state.failNext} onCheckedChange={value => setMockFailNext(Boolean(value))} data-testid="mock-debug-fail-next" />下一次请求返回 503</label>
            <div className="flex flex-wrap gap-2" data-testid="mock-debug-actions"><Button variant="outline" size="sm" onClick={() => resetMockState()} data-testid="mock-debug-reset"><RotateCcw size={14} /> 重置数据</Button><Button variant="outline" size="sm" onClick={() => clearMockRequests()} data-testid="mock-debug-clear"><Trash2 size={14} /> 清空记录</Button></div>
            <div className="flex items-center justify-between text-xs text-neutral-500" data-testid="mock-debug-request-summary"><span>最近请求（{state.requests.length}）</span><span>本地存储隔离</span></div>
            <div className="max-h-52 overflow-auto rounded-md bg-neutral-950 p-2 font-mono text-[11px] text-neutral-100" data-testid="mock-debug-request-list">
              {state.requests.length ? state.requests.map((request, index) => <RequestRow request={request} key={`${request.at}-${index}`} data-testid={`mock-request-${index}`} />) : <div className="p-3 text-neutral-400" data-testid="mock-debug-empty">暂无请求</div>}
            </div>
          </div>
        </Card>
      )}
    </aside>
  )
}

function RequestRow({ request, 'data-testid': testId }: { request: MockRequest; 'data-testid': string }) {
  return <div className="border-b border-white/10 px-2 py-1.5 last:border-0" data-testid={testId}><div className="flex items-center justify-between gap-2"><span className={request.status >= 400 ? 'text-red-300' : 'text-emerald-300'}>{request.method} {request.status}</span><span className="text-neutral-400">{request.duration}ms</span></div><div className="truncate text-neutral-300">{request.url}</div>{request.error ? <div className="text-red-300">{request.error}</div> : null}</div>
}

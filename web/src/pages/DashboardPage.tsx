import { useEffect, useState } from 'react'
import { Boxes, Globe, Network, Play, Power, RefreshCw, RotateCw, ScrollText, ShieldCheck, type LucideIcon } from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { cn } from '../lib/utils'
import { getJson, postJson } from '../lib/api'
import type { ProcessStatus, RuntimeLogSummary, RuntimeStatus } from '../types'

type ProcessName = 'nginx' | 'xray'
type ProcessAction = 'start' | 'stop' | 'restart'

const actionLabels: Record<ProcessAction, string> = {
  start: '启动',
  stop: '停止',
  restart: '重启',
}

export function DashboardPage() {
  const [status, setStatus] = useState<RuntimeStatus>()
  const [busy, setBusy] = useState<string>()
  const [lastUpdated, setLastUpdated] = useState<Date>()
  const [xrayLogs, setXrayLogs] = useState<string[]>([])
  const [logsOpen, setLogsOpen] = useState(false)
  const [logsLoading, setLogsLoading] = useState(false)

  const load = () =>
    getJson<RuntimeStatus>('runtime/status').then((nextStatus) => {
      setStatus(nextStatus)
      setLastUpdated(new Date())
    })

  useEffect(() => {
    void load()
  }, [])

  async function apply() {
    setBusy('apply')
    try {
      await postJson('runtime/apply')
      toast.success('配置已应用')
    } catch {
      toast.error('配置应用失败，请查看日志')
    } finally {
      setBusy(undefined)
      void load()
    }
  }

  async function control(process: ProcessName, action: ProcessAction) {
    const key = `${process}-${action}`
    setBusy(key)
    try {
      await postJson(`runtime/${process}/${action}`)
      toast.success(`${process} ${actionLabels[action]}完成`)
    } catch {
      toast.error(`${process} ${actionLabels[action]}失败`)
    } finally {
      setBusy(undefined)
      void load()
    }
  }

  async function showXrayLogs() {
    setLogsOpen(true)
    setLogsLoading(true)
    try {
      const result = await getJson<RuntimeLogSummary>('runtime/xray/logs')
      setXrayLogs(result.logs ?? [])
    } catch {
      toast.error('读取 Xray 日志失败')
    } finally {
      setLogsLoading(false)
    }
  }

  return (
    <div className="space-y-7" data-testid="dashboard-page">
      <header className="flex flex-wrap items-start justify-between gap-4" data-testid="dashboard-header">
        <div className="min-w-0 flex-1 basis-72" data-testid="dashboard-heading-group">
          <h1 className="text-3xl font-semibold leading-tight tracking-[-0.06em] text-[#171717] text-balance" data-testid="dashboard-title">
            Dashboard
          </h1>
        </div>

        <div
          className="flex min-h-12 flex-wrap items-center gap-3 rounded-xl bg-white px-2 py-2 shadow-[var(--shadow-border)]"
          data-testid="dashboard-actions"
        >
          <Button variant="outline" onClick={apply} disabled={busy === 'apply'} data-testid="dashboard-apply-button">
            <RefreshCw size={16} aria-hidden="true" />
            应用配置
          </Button>
          <Button variant="outline" onClick={() => void load()} data-testid="dashboard-refresh-button">
            <RotateCw size={16} aria-hidden="true" />
            刷新状态
          </Button>
          <div className="h-6 w-px bg-neutral-200" aria-hidden="true" data-testid="dashboard-action-divider" />
          <div className="px-2 text-sm tabular-nums text-neutral-500" data-testid="dashboard-last-updated">
            最后更新：{lastUpdated ? formatDateTime(lastUpdated) : '-'}
          </div>
        </div>
      </header>

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4" data-testid="dashboard-metrics">
        <Metric
          label="域名"
          value={status?.domainCount}
          caption="已配置域名"
          Icon={Globe}
          data-testid="dashboard-metric-domains"
        />
        <Metric
          label="证书"
          value={status?.certificateCount}
          caption={status?.expiringCertificateCount ? `${status.expiringCertificateCount} 即将到期` : '证书正常'}
          Icon={ShieldCheck}
          data-testid="dashboard-metric-certificates"
        />
        <Metric
          label="反代规则"
          value={status?.reverseProxyCount}
          caption="已配置规则"
          Icon={Network}
          data-testid="dashboard-metric-reverse-proxies"
        />
        <Metric
          label="代理入口"
          value={status?.inboundCount}
          caption="代理入口数量"
          Icon={Boxes}
          data-testid="dashboard-metric-inbounds"
        />
      </section>

      <section className="space-y-5" data-testid="dashboard-service-section">
        <div className="space-y-2" data-testid="dashboard-service-heading">
          <h2 className="flex items-center gap-2 text-xl font-semibold tracking-[-0.04em] text-[#171717]" data-testid="dashboard-service-title">
            <span
              className="grid size-5 place-items-center rounded-md shadow-[var(--shadow-border-subtle)]"
              aria-hidden="true"
            >
              <RefreshCw size={14} />
            </span>
            服务与进程
          </h2>
          <p className="text-sm text-neutral-500">
            当前系统中核心服务与进程的运行状态。
          </p>
        </div>

        <div className="grid gap-4 xl:grid-cols-2" data-testid="dashboard-processes">
          <ProcessCard
            title="Nginx"
            process="nginx"
            status={status?.nginx}
            listen={formatPorts(status?.nginxPorts)}
            busy={busy}
            onAction={control}
            data-testid="dashboard-process-nginx"
          />
          <ProcessCard
            title="Xray"
            process="xray"
            status={status?.xray}
            listen="-"
            busy={busy}
            onAction={control}
            onLogs={showXrayLogs}
            data-testid="dashboard-process-xray"
          />
        </div>
      </section>

      {logsOpen ? (
        <Dialog open={logsOpen} onOpenChange={setLogsOpen}>
          <DialogContent className="max-w-3xl p-0" data-testid="dashboard-logs-dialog">
            <DialogHeader className="px-4 pt-4">
              <DialogTitle className="flex items-center gap-2">
                <ScrollText size={18} aria-hidden="true" />
                Xray 日志
              </DialogTitle>
            </DialogHeader>
            <pre className="m-0 min-h-64 max-h-[70vh] overflow-auto rounded-b-xl bg-neutral-950 p-4 text-xs leading-5 text-neutral-100 whitespace-pre-wrap">
              {logsLoading ? '加载中…' : xrayLogs.length ? xrayLogs.join('\n') : '暂无日志'}
            </pre>
          </DialogContent>
        </Dialog>
      ) : null}
    </div>
  )
}

function ProcessCard({
  title,
  process,
  status,
  listen,
  busy,
  onAction,
  onLogs,
  'data-testid': dataTestId,
}: {
  title: string
  process: ProcessName
  status?: ProcessStatus
  listen: string
  busy?: string
  onAction: (process: ProcessName, action: ProcessAction) => Promise<void>
  onLogs?: () => Promise<void>
  'data-testid'?: string
}) {
  const running = Boolean(status?.running)

  return (
    <Card className="rounded-xl bg-white p-7 shadow-[var(--shadow-border)]" data-testid={dataTestId}>
      <CardHeader className="grid-cols-[auto_1fr_auto] gap-4 px-0" data-testid={`${dataTestId}-header`}>
        <ProcessLogo process={process} data-testid={`${dataTestId}-logo`} />
        <div className="min-w-0" data-testid={`${dataTestId}-identity`}>
          <CardTitle className="text-xl font-semibold tracking-[-0.04em]" data-testid={`${dataTestId}-title`}>{title}</CardTitle>
          <CardDescription className="truncate text-sm" data-testid={`${dataTestId}-path`}>{status?.path || '-'}</CardDescription>
        </div>
        <RuntimeBadge running={running} data-testid={`${dataTestId}-status`} />
      </CardHeader>

      <div className={`mt-6 grid gap-3 ${onLogs ? 'sm:grid-cols-4' : 'sm:grid-cols-3'}`} data-testid={`${dataTestId}-actions`}>
        <Button className="h-10 bg-neutral-50 text-neutral-900 hover:bg-neutral-100" variant="secondary" disabled={busy === `${process}-start`} onClick={() => onAction(process, 'start')} data-testid={`${dataTestId}-start`}>
          <Play size={16} aria-hidden="true" />
          启动
        </Button>
        <Button className="h-10 bg-neutral-50 text-neutral-900 hover:bg-neutral-100" variant="secondary" disabled={busy === `${process}-stop`} onClick={() => onAction(process, 'stop')} data-testid={`${dataTestId}-stop`}>
          <Power size={16} aria-hidden="true" />
          停止
        </Button>
        <Button className="h-10 bg-neutral-50 text-neutral-900 hover:bg-neutral-100" variant="secondary" disabled={busy === `${process}-restart`} onClick={() => onAction(process, 'restart')} data-testid={`${dataTestId}-restart`}>
          <RotateCw size={16} aria-hidden="true" />
          重启
        </Button>
        {onLogs ? (
          <Button className="h-10 bg-neutral-50 text-neutral-900 hover:bg-neutral-100" variant="secondary" onClick={onLogs} data-testid={`${dataTestId}-logs`}>
            <ScrollText size={16} aria-hidden="true" />
            日志
          </Button>
        ) : null}
      </div>

      <dl className="mt-7 grid gap-0 text-sm" data-testid={`${dataTestId}-details`}>
        <StatusRow label="监听地址" value={listen} data-testid={`${dataTestId}-listen`} />
        <StatusRow label="启动时间" value={status?.startedAt || '-'} data-testid={`${dataTestId}-started-at`} />
        <StatusRow label="最后错误" value={status?.lastError || '-'} data-testid={`${dataTestId}-last-error`} />
      </dl>
    </Card>
  )
}

function Metric({
  label,
  value,
  caption,
  Icon,
  'data-testid': dataTestId,
}: {
  label: string
  value?: number
  caption: string
  Icon: LucideIcon
  'data-testid'?: string
}) {
  return (
    <Card className="min-h-28 rounded-xl bg-white p-6 shadow-[var(--shadow-border)]" data-testid={dataTestId}>
      <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-5" data-testid={`${dataTestId}-content`}>
        <MetricIcon Icon={Icon} data-testid={`${dataTestId}-icon`} />
        <div className="min-w-0" data-testid={`${dataTestId}-body`}>
          <div className="text-sm font-medium text-neutral-600" data-testid={`${dataTestId}-label`}>{label}</div>
          <div className="mt-2 text-sm text-neutral-500" data-testid={`${dataTestId}-caption`}>{caption}</div>
        </div>
        <div className="justify-self-end text-3xl font-semibold tracking-[-0.05em] text-[#171717] tabular-nums" data-testid={`${dataTestId}-value`}>
          {String(value ?? '-')}
        </div>
      </div>
    </Card>
  )
}

function MetricIcon({ Icon, 'data-testid': dataTestId }: { Icon: LucideIcon; 'data-testid'?: string }) {
  return (
    <div className="grid size-16 place-items-center rounded-2xl bg-neutral-50 text-neutral-900 shadow-[var(--shadow-border)]" data-testid={dataTestId}>
      <Icon size={28} strokeWidth={1.8} aria-hidden="true" />
    </div>
  )
}

function StatusRow({ label, value, 'data-testid': dataTestId }: { label: string; value: string; 'data-testid'?: string }) {
  return (
    <div className="grid grid-cols-[96px_1fr] gap-5 py-4 shadow-[inset_0_1px_0_rgba(0,0,0,0.08)] first:pt-0 first:shadow-none last:pb-0" data-testid={dataTestId}>
      <dt className="text-neutral-500" data-testid={`${dataTestId}-label`}>{label}</dt>
      <dd className="min-w-0 break-words tabular-nums text-neutral-900" data-testid={`${dataTestId}-value`}>{value}</dd>
    </div>
  )
}

function RuntimeBadge({ running, 'data-testid': dataTestId }: { running?: boolean; 'data-testid'?: string }) {
  return (
    <Badge
      variant="secondary"
      className={cn('rounded-full px-4 py-1 text-sm font-medium', running ? 'bg-secondary text-secondary-foreground' : 'bg-neutral-100 text-neutral-500')}
      data-testid={dataTestId}
    >
      {running ? '运行中' : '已停止'}
    </Badge>
  )
}

function formatPorts(ports?: number[]) {
  return ports?.length ? ports.join(', ') : '-'
}

function formatDateTime(date: Date) {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(date)
}

function ProcessLogo({ process, 'data-testid': dataTestId }: { process: ProcessName; 'data-testid'?: string }) {
  const src = process === 'nginx' ? '/svg/nginx.svg' : '/svg/xray.svg'
  const alt = process === 'nginx' ? 'Nginx' : 'Xray'

  return (
    <div className="grid size-16 place-items-center rounded-2xl bg-white p-3 shadow-[var(--shadow-border)]" data-testid={dataTestId}>
      <img src={src} alt={alt} className="size-full object-contain" />
    </div>
  )
}

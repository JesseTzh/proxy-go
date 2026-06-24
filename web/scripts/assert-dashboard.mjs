import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const source = readFileSync(resolve(__dirname, '../src/pages/DashboardPage.tsx'), 'utf8')

const expectations = [
  ['dashboard-last-updated', 'top status capsule with last updated time'],
  ['dashboard-service-section', 'service section'],
  ['dashboard-metric-domains', 'domain metric card'],
  ['dashboard-metric-certificates', 'certificate metric card'],
  ['dashboard-metric-reverse-proxies', 'reverse proxy metric card'],
  ['dashboard-metric-inbounds', 'proxy entry metric card'],
  ['/svg/nginx.svg', 'nginx process svg asset'],
  ['/svg/xray.svg', 'xray process svg asset'],
  ['`${dataTestId}-icon`', 'dynamic metric icon test id'],
  ['`${dataTestId}-listen`', 'dynamic process listen row test id'],
  ['grid-cols-[auto_minmax(0,1fr)_auto]', 'compact metric card with right-aligned value'],
  ['ProcessLogo', 'process logo component'],
  ['MetricIcon', 'metric icon component'],
  ['text-neutral-900', 'neutral metric icon color'],
  ['bg-neutral-50', 'neutral metric icon background'],
]

const missing = expectations.filter(([needle]) => !source.includes(needle))
const legacyAttribute = 'data-' + 'test-id'
const oldAttributeCount = source.split(legacyAttribute).length - 1
const removedDescription = '集中查看服务、资源与运行进程。' + '页面保留必要信息，避免重复状态噪音。'

if (missing.length > 0) {
  console.error('Dashboard redesign assertions failed:')
  for (const [, label] of missing) {
    console.error(`- Missing ${label}`)
  }
  process.exit(1)
}

if (oldAttributeCount > 0) {
  console.error(`Dashboard redesign assertions failed: found ${oldAttributeCount} legacy test id attributes`)
  process.exit(1)
}

if (source.includes(removedDescription)) {
  console.error('Dashboard redesign assertions failed: removed dashboard description is still present')
  process.exit(1)
}

const forbiddenUiColorClasses = [
  'bg-emerald-',
  'text-emerald-',
  'bg-blue-',
  'text-blue-',
  'bg-violet-',
  'text-violet-',
  'bg-orange-',
  'text-orange-',
  'text-cyan-',
]

const forbiddenColors = forbiddenUiColorClasses.filter((needle) => source.includes(needle))
if (forbiddenColors.length > 0) {
  console.error('Dashboard redesign assertions failed: non-DESIGN.md UI color classes remain:')
  for (const needle of forbiddenColors) {
    console.error(`- ${needle}`)
  }
  process.exit(1)
}

const forbiddenMetricAssets = [
  '/svg/domain.svg',
  '/svg/ssl.svg',
  '/svg/reverse-proxy.svg',
  '/svg/vpn.svg',
]

const remainingMetricAssets = forbiddenMetricAssets.filter((needle) => source.includes(needle))
if (remainingMetricAssets.length > 0) {
  console.error('Dashboard redesign assertions failed: metric icons still use standalone svg assets:')
  for (const needle of remainingMetricAssets) {
    console.error(`- ${needle}`)
  }
  process.exit(1)
}

const forbiddenServiceTestIds = [
  'data-testid="dashboard-service-icon"',
  'data-testid="dashboard-service-description"',
]

const remainingServiceTestIds = forbiddenServiceTestIds.filter((needle) => source.includes(needle))
if (remainingServiceTestIds.length > 0) {
  console.error('Dashboard redesign assertions failed: removed service test ids are still present:')
  for (const needle of remainingServiceTestIds) {
    console.error(`- ${needle}`)
  }
  process.exit(1)
}

console.log('Dashboard redesign assertions passed')

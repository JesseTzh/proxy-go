import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const formFieldSource = readFileSync(resolve(__dirname, '../src/components/FormField.tsx'), 'utf8')
const vlessSource = readFileSync(resolve(__dirname, '../src/pages/VlessPage.tsx'), 'utf8')

const expectations = [
  [formFieldSource, 'description?: string', 'FormField description prop'],
  [formFieldSource, 'HelpCircle', 'help icon'],
  [formFieldSource, '`${dataTestId}-help`', 'field help test id'],
  [formFieldSource, '`${dataTestId}-tooltip`', 'field tooltip test id'],
  [vlessSource, 'VLESS XHTTP Reality', 'xhttp reality default name'],
  [vlessSource, 'label="客户端连接域名"', 'client connection domain label'],
  [vlessSource, '客户端实际连接的域名', 'client domain explanation'],
  [vlessSource, '不会作为 REALITY serverName/sni', 'client domain does not become reality server name explanation'],
  [vlessSource, '当前仅支持使用 443 端口作为公网入口', 'public entry note'],
  [vlessSource, 'label="REALITY 握手服务器"', 'reality handshake server label'],
  [vlessSource, 'REALITY 客户端使用的伪装 SNI', 'handshake server explanation'],
  [vlessSource, '例如 apple.com', 'apple handshake server example'],
  [vlessSource, '普通 HTTPS 固定回落到内部 Nginx', 'managed https fallback explanation'],
  [vlessSource, 'label="XHTTP 路径"', 'xhttp path label'],
]

const missing = expectations.filter(([source, needle]) => !source.includes(needle))
if (missing.length > 0) {
  console.error('Inbound domain help assertions failed:')
  for (const [, , label] of missing) {
    console.error(`- Missing ${label}`)
  }
  process.exit(1)
}

const forbidden = [
  'vless-reality-vision',
  'Reality Vision',
  'label="模板"',
  'label="Xray 监听端口"',
  'label="安全层"',
  'TLS 模式不会使用该字段',
  'Nginx 站点并转发对应路径',
  'label="公网入口"',
  'inbound-public-entry-field',
  'label="REALITY 握手端口"',
  'realityHandshakePort',
  'label="XHTTP 模式"',
  'inbound-xhttp-mode-field',
  'label="最大时间差"',
  'inbound-max-time-diff-field',
  'inbound-enabled-field',
]
const remaining = forbidden.filter((needle) => vlessSource.includes(needle))
if (remaining.length > 0) {
  console.error('Inbound domain help assertions failed: removed legacy UI remains:')
  for (const needle of remaining) {
    console.error(`- ${needle}`)
  }
  process.exit(1)
}

console.log('Inbound domain help assertions passed')

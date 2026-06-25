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
  [vlessSource, 'VLESS Reality Vision', 'vision default name'],
  [vlessSource, 'AnyTLS', 'anytls option'],
  [vlessSource, 'label="客户端连接域名"', 'client connection domain label'],
  [vlessSource, 'Reality Vision 客户端实际连接的域名', 'vision domain explanation'],
  [vlessSource, 'AnyTLS 使用该域名作为 TLS 证书域名和 SNI 分流入口', 'anytls domain explanation'],
  [vlessSource, '公网 443 由 Nginx stream 统一监听', 'public entry note'],
  [vlessSource, 'label="REALITY 握手服务器"', 'reality handshake server label'],
  [vlessSource, 'Reality Vision 客户端使用的伪装 SNI', 'handshake server explanation'],
  [vlessSource, '例如 apple.com', 'apple handshake server example'],
  [vlessSource, '不要填写已托管域名', 'managed domain conflict explanation'],
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
  'label="模板"',
  'label="' + 'X' + 'ray 监听端口"',
  'X' + 'ray',
  'label="安全层"',
  'TLS 模式不会使用该字段',
  'Nginx 站点并转发对应路径',
  'label="公网入口"',
  'inbound-public-entry-field',
  'label="REALITY 握手端口"',
  'realityHandshakePort',
  'label="' + 'X' + 'HTTP 模式"',
  'label="' + 'X' + 'HTTP 路径"',
  'inbound-' + 'x' + 'http-section',
  'inbound-' + 'x' + 'http-mode-field',
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

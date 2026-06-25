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
  [vlessSource, 'XHTTP REALITY 的 serverName', 'xhttp server name explanation'],
  [vlessSource, 'label="公网入口"', 'public entry label'],
  [vlessSource, 'Xray 直接监听公网 HTTPS 端口 443', 'public entry explanation'],
  [vlessSource, 'Nginx 不再转发 XHTTP 流量', 'nginx no xhttp forwarding explanation'],
  [vlessSource, 'label="REALITY 握手服务器"', 'reality handshake server label'],
  [vlessSource, 'REALITY 伪装握手的目标站点', 'handshake server explanation'],
  [vlessSource, 'label="REALITY 握手端口"', 'reality handshake port label'],
  [vlessSource, 'REALITY 伪装握手服务器的端口', 'handshake port explanation'],
  [vlessSource, '通常是 443', 'usual handshake port explanation'],
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

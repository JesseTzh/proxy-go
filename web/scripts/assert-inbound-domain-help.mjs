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
  [vlessSource, 'label="客户端连接域名"', 'client connection domain label'],
  [vlessSource, '客户端实际连接的域名', 'client domain explanation'],
  [vlessSource, 'XHTTP 入口还会按这个域名匹配 Nginx 站点', 'xhttp routing explanation'],
  [vlessSource, 'label="REALITY 握手服务器"', 'reality handshake server label'],
  [vlessSource, 'REALITY 伪装握手的目标站点', 'handshake server explanation'],
  [vlessSource, 'TLS 模式不会使用该字段', 'tls unused explanation'],
]

const missing = expectations.filter(([source, needle]) => !source.includes(needle))
if (missing.length > 0) {
  console.error('Inbound domain help assertions failed:')
  for (const [, , label] of missing) {
    console.error(`- Missing ${label}`)
  }
  process.exit(1)
}

console.log('Inbound domain help assertions passed')

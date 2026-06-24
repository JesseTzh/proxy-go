import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const source = readFileSync(resolve(__dirname, '../src/pages/LoginPage.tsx'), 'utf8')

const expectations = [
  ['Proxy-Go', 'centered product name'],
  ['text-center', 'centered product name alignment'],
  ['data-testid="login-page"', 'login page test id'],
  ['data-testid="login-form"', 'login form test id'],
  ['data-testid="login-password-input"', 'password input test id'],
  ['data-testid="login-submit-button"', 'submit button test id'],
]

const missing = expectations.filter(([needle]) => !source.includes(needle))
if (missing.length > 0) {
  console.error('Login assertions failed:')
  for (const [, label] of missing) {
    console.error(`- Missing ${label}`)
  }
  process.exit(1)
}

const forbidden = [
  'Lock',
  '<Lock',
  'proxy-go',
  '管理面板仅使用密码登录。',
]

const remaining = forbidden.filter((needle) => source.includes(needle))
if (remaining.length > 0) {
  console.error('Login assertions failed: removed login content is still present:')
  for (const needle of remaining) {
    console.error(`- ${needle}`)
  }
  process.exit(1)
}

console.log('Login assertions passed')

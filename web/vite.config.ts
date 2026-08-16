import { execFileSync } from 'node:child_process'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

function createVersion() {
  const now = new Date()
  const dateParts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: '2-digit',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(now)
  const part = (type: Intl.DateTimeFormatPartTypes) =>
    dateParts.find((item) => item.type === type)?.value ?? '00'
  const date = [part('year'), part('month'), part('day'), part('hour'), part('minute')].join('-')

  let gitSha = process.env.PROXY_GO_GIT_SHA?.trim().slice(0, 7)
  if (!gitSha) {
    try {
      gitSha = execFileSync('git', ['rev-parse', '--short', 'HEAD'], {
        encoding: 'utf8',
      }).trim()
    } catch {
      gitSha = 'unknown'
    }
  }

  return `ver:${date}-${gitSha}`
}

export default defineConfig({
  define: {
    __APP_VERSION__: JSON.stringify(createVersion()),
  },
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': new URL('./src', import.meta.url).pathname,
    },
  },
  server: { proxy: { '/api': 'http://127.0.0.1:30080' } }
})

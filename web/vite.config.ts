import { execFileSync } from 'node:child_process'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

function createVersion() {
  const now = new Date()
  const date = [
    String(now.getFullYear()).slice(-2),
    String(now.getMonth() + 1).padStart(2, '0'),
    String(now.getDate()).padStart(2, '0'),
    String(now.getHours()).padStart(2, '0'),
    String(now.getMinutes()).padStart(2, '0'),
  ].join('-')

  let gitSha = 'unknown'
  try {
    gitSha = execFileSync('git', ['rev-parse', '--short', 'HEAD'], {
      encoding: 'utf8',
    }).trim()
  } catch {
    // Keep builds working when the source is provided without Git metadata.
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

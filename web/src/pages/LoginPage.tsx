import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { postJson } from '../lib/api'

export function LoginPage({onLogin}:{onLogin:()=>void}) {
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  async function submit(e: React.FormEvent){ e.preventDefault(); setLoading(true); try { await postJson('auth/login', {password}, { loading: false }); onLogin() } catch { /* global error dialog handles feedback */ } finally { setLoading(false) } }
  return (
    <div className="min-h-screen grid place-items-center p-6" data-testid="login-page">
      <div className="w-full max-w-sm" data-testid="login-panel">
        <Card className="w-full p-6" data-testid="login-card">
          <form onSubmit={submit} className="space-y-4" data-testid="login-form">
            <div className="text-center text-xl font-semibold tracking-[-0.03em]" data-testid="login-product-name">Proxy-Go</div>
            <Input type="password" placeholder="管理密码…" value={password} onChange={e=>setPassword(e.target.value)} data-testid="login-password-input" />
            <Button className="w-full" disabled={loading} type="submit" data-testid="login-submit-button">{loading ? '登录中…' : '登录'}</Button>
          </form>
        </Card>
        <div className="mt-3 text-center text-xs text-muted-foreground" data-testid="login-version">{__APP_VERSION__}</div>
      </div>
    </div>
  )
}

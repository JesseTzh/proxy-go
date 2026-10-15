import type { AuditLog, Domain, InboundShare, ProxyInbound, ReverseProxy, RuntimeConfigSnapshot, RuntimeLogSummary, RuntimeStatus } from '../types'

export type MockRequest = { method: string; url: string; body?: unknown; status: number; duration: number; at: string; error?: string }

type MockState = {
  authenticated: boolean
  nextId: number
  requests: MockRequest[]
  failNext: boolean
  delay: number
  domains: Domain[]
  reverseProxies: ReverseProxy[]
  inbounds: ProxyInbound[]
  settings: Record<string, unknown>
  runtime: RuntimeStatus
  auditLogs: AuditLog[]
}

const storageKey = 'proxy-go:mock:v1'
const listeners = new Set<() => void>()

export const isMockMode = import.meta.env.VITE_MOCK_API === 'true'

const iso = (minutes = 0) => new Date(Date.now() - minutes * 60_000).toISOString()

function initialState(): MockState {
  const domains: Domain[] = [
    { id: 1, domain: 'panel.example.com', status: 'enabled', remark: '管理面板', createdAt: iso(360), certificate: { id: 1, primaryDomain: 'panel.example.com', status: 'issued', expiresAt: '2027-06-30', autoRenew: true } },
    { id: 2, domain: 'edge.example.com', status: 'enabled', remark: '代理入口', createdAt: iso(240), certificate: { id: 2, primaryDomain: 'edge.example.com', status: 'issued', expiresAt: '2027-04-15', autoRenew: true } },
  ]
  return {
    authenticated: true, nextId: 20, requests: [], failNext: false, delay: 220,
    domains,
    reverseProxies: [{ id: 1, domainId: 1, domain: domains[0], targetScheme: 'http', targetHost: '127.0.0.1', targetPort: 8080, preserveHost: true, webSocket: true, passRealIp: true, enabled: true, remark: '管理后台' }],
    inbounds: [{ id: 1, name: '主入口', template: 'vless-reality-vision', protocol: 'vless', domainId: 2, domain: domains[1], network: 'tcp', security: 'reality', flow: 'xtls-rprx-vision', routeSni: 'www.apple.com', listenAddr: '0.0.0.0', listenPort: 443, realityHandshakeServer: 'www.apple.com', enabled: true }],
    settings: { initialPortEnabled: true, managementDomain: domains[0].domain, acmeEmail: 'ops@example.com', runtimeConfigStatus: 'applied', paths: { data: '/var/lib/proxy-go', config: '/etc/proxy-go/config.yml' }, versions: { proxyGo: 'mock 0.1.0', nginx: '1.27.0', singBox: '1.12.0' } },
    runtime: { proxyGo: { running: true }, nginx: { name: 'nginx', path: '/usr/sbin/nginx', running: true, startedAt: iso(120) }, singBox: { name: 'sing-box', path: '/usr/local/bin/sing-box', running: true, startedAt: iso(118) }, goInternalAddr: '127.0.0.1:30080', nginxPublicHttpPort: 80, nginxPublicHttpsPort: 443, nginxManagedHttpsAddr: '127.0.0.1:30443', singBoxInboundListen: '0.0.0.0:443', singBoxDebugEnabled: false, domainCount: domains.length, certificateCount: 2, reverseProxyCount: 1, inboundCount: 1, expiringCertificateCount: 0 },
    auditLogs: [{ id: 1, action: 'mock_bootstrap', resourceType: 'runtime', resourceId: '', detail: '{"mode":"mock"}', ip: '127.0.0.1', userAgent: 'Proxy-Go Mock', createdAt: iso(2) }],
  }
}

let state = load()
function load(): MockState { try { const raw = localStorage.getItem(storageKey); return raw ? JSON.parse(raw) as MockState : initialState() } catch { return initialState() } }
function save() { localStorage.setItem(storageKey, JSON.stringify(state)); listeners.forEach(listener => listener()) }
export function subscribeMock(listener: () => void) { listeners.add(listener); return () => { listeners.delete(listener) } }
export function getMockState() { return state }
export function resetMockState() { state = initialState(); save() }
export function setMockDelay(delay: number) { state.delay = Math.max(0, Math.min(3000, delay)); save() }
export function setMockFailNext(value: boolean) { state.failNext = value; save() }
export function setMockAuthenticated(value: boolean) { state.authenticated = value; save(); window.dispatchEvent(new CustomEvent('proxy-go:mock-auth', { detail: value })) }
export function clearMockRequests() { state.requests = []; save() }

function parseBody(body?: BodyInit | null): unknown { if (typeof body !== 'string' || !body) return undefined; try { return JSON.parse(body) } catch { return body } }
function record(method: string, url: string, status: number, started: number, error?: string) { state.requests = [{ method, url, status, duration: Date.now() - started, at: new Date().toISOString(), error }, ...state.requests].slice(0, 40); save() }
function ok(data: unknown) { return new Response(JSON.stringify({ ok: true, data }), { status: 200, headers: { 'content-type': 'application/json' } }) }
function fail(message: string, status = 500) { return new Response(JSON.stringify({ ok: false, error: { message } }), { status, headers: { 'content-type': 'application/json' } }) }

export async function mockFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const started = Date.now(); const method = (init?.method ?? 'GET').toUpperCase(); const url = String(input).replace(/^.*\/api\/?/, '').replace(/^\//, ''); const body = parseBody(init?.body)
  await new Promise(resolve => setTimeout(resolve, state.delay))
  if (state.failNext) { state.failNext = false; record(method, url, 503, started, '模拟故障：下一次请求失败'); return fail('模拟故障：下一次请求失败', 503) }
  let response: Response
  try { response = route(method, url, body); record(method, url, response.status, started); return response } catch (error) { const message = error instanceof Error ? error.message : 'Mock 请求失败'; record(method, url, 500, started, message); return fail(message) }
}

function route(method: string, url: string, body: unknown): Response {
  if (url === 'auth/me' && method === 'GET') return state.authenticated ? ok({ authenticated: true }) : fail('未登录', 401)
  if (url === 'auth/login' && method === 'POST') { state.authenticated = true; save(); return ok({ authenticated: true }) }
  if (url === 'auth/logout' && method === 'POST') { state.authenticated = false; save(); return ok(null) }
  if (url === 'domains' && method === 'GET') return ok(state.domains)
  if (url === 'domains' && method === 'POST') { const value = body as { domain?: string }; const domain: Domain = { id: state.nextId++, domain: value.domain || 'new.example.com', status: 'enabled', remark: '', createdAt: new Date().toISOString() }; state.domains.push(domain); save(); return ok(domain) }
  const domainMatch = url.match(/^domains\/(\d+)(?:\/(.*))?$/); if (domainMatch) return domainRoute(Number(domainMatch[1]), domainMatch[2], method)
  if (url === 'reverse-proxies' && method === 'GET') return ok(state.reverseProxies)
  if (url === 'reverse-proxies' && method === 'POST') { const item = { ...(body as object), id: state.nextId++, domain: state.domains.find(x => x.id === Number((body as { domainId: number }).domainId)) } as ReverseProxy; state.reverseProxies.push(item); save(); return ok(item) }
  const reverseMatch = url.match(/^reverse-proxies\/(\d+)$/); if (reverseMatch) { const index = state.reverseProxies.findIndex(x => x.id === Number(reverseMatch[1])); if (method === 'DELETE') { state.reverseProxies.splice(index, 1); save(); return ok(null) } if (method === 'PUT') { state.reverseProxies[index] = { ...state.reverseProxies[index], ...(body as object) } as ReverseProxy; save(); return ok(state.reverseProxies[index]) } }
  if (url === 'inbounds' && method === 'GET') return ok(state.inbounds)
  if (url === 'inbounds' && method === 'POST') { const value = body as ProxyInbound; const item = { ...value, id: state.nextId++, protocol: value.template?.startsWith('vless') ? 'vless' : value.template, domain: state.domains.find(x => x.id === value.domainId), enabled: true } as ProxyInbound; state.inbounds.push(item); save(); return ok(item) }
  const inboundMatch = url.match(/^inbounds\/(\d+)(?:\/(.*))?$/); if (inboundMatch) return inboundRoute(Number(inboundMatch[1]), inboundMatch[2], method, body)
  if (url === 'settings' && method === 'GET') return ok({ settings: state.settings, paths: state.settings.paths, versions: state.settings.versions })
  if (url === 'settings' && method === 'PUT') { state.settings = { ...state.settings, ...(body as object) }; save(); return ok(null) }
  if (url === 'init/disable-initial-port' && method === 'POST') { state.settings.initialPortEnabled = false; save(); return ok({ message: 'mock updated' }) }
  if (url === 'runtime/status' && method === 'GET') return ok(state.runtime)
  if (url.match(/^runtime\/(nginx|sing-box)\/(start|stop|restart)$/) && method === 'POST') { const match = url.match(/^runtime\/(nginx|sing-box)\/(start|stop|restart)$/)!; const target = match[1] === 'nginx' ? state.runtime.nginx : state.runtime.singBox; target.running = match[2] !== 'stop'; target.startedAt = new Date().toISOString(); save(); return ok(null) }
  if (url === 'runtime/sing-box/logs' && method === 'GET') return ok({ logs: ['2026-10-15T10:10:02Z info sing-box started', '2026-10-15T10:10:04Z info inbound listening on 0.0.0.0:443'] } satisfies RuntimeLogSummary)
  if (url === 'runtime/nginx/config' && method === 'GET') return ok({ path: '/etc/nginx/conf.d/proxy-go.conf', content: 'server {\n  listen 443 ssl;\n  proxy_pass http://127.0.0.1:30080;\n}' } satisfies RuntimeConfigSnapshot)
  if (url.match(/^runtime\/sing-box\/debug\/(enable|disable)$/) && method === 'POST') { state.runtime.singBoxDebugEnabled = url.endsWith('enable'); save(); return ok(null) }
  if (url === 'audit-logs' && method === 'GET') return ok(state.auditLogs)
  return ok(null)
}
function domainRoute(id: number, action: string | undefined, method: string): Response { const item = state.domains.find(x => x.id === id); if (!item) return fail('域名不存在', 404); if (!action && method === 'DELETE') { state.domains = state.domains.filter(x => x.id !== id); save(); return ok(null) } if (action === 'management' && method === 'POST') { state.settings.managementDomain = item.domain; save(); return ok(null) } if (action === 'dns-check' && method === 'POST') return ok({ reachable: true, addresses: ['203.0.113.10'] }) ; if (action?.startsWith('certificate') && method === 'POST') { item.certificate = { id: id, primaryDomain: item.domain, status: 'issued', expiresAt: '2027-12-31', autoRenew: true }; state.runtime.certificateCount = state.domains.filter(x => x.certificate).length; save(); return ok(item.certificate) } if (action === 'certificate' && method === 'DELETE') { delete item.certificate; save(); return ok(null) } return ok(item) }
function inboundRoute(id: number, action: string | undefined, method: string, body: unknown): Response { const index = state.inbounds.findIndex(x => x.id === id); const item = state.inbounds[index]; if (!item) return fail('代理入口不存在', 404); if (!action && method === 'DELETE') { state.inbounds.splice(index, 1); state.runtime.inboundCount = state.inbounds.length; save(); return ok(null) } if (!action && method === 'PUT') { state.inbounds[index] = { ...item, ...(body as object) } as ProxyInbound; save(); return ok(state.inbounds[index]) } if (action === 'enable' || action === 'disable') { item.enabled = action === 'enable'; save(); return ok(item) } if (action === 'config') return ok({ type: item.template, listen: `${item.listenAddr}:${item.listenPort}`, server: item.realityHandshakeServer }) ; if (action === 'share') return ok({ name: item.name, domain: item.domain?.domain ?? 'edge.example.com', uri: 'vless://mock-uuid@edge.example.com:443?security=reality#Proxy-Go' } satisfies InboundShare); return ok(item) }

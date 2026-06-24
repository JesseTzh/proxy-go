import { useEffect, useState } from 'react'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { getJson } from '../lib/api'
import type { AuditLog } from '../types'

export function AuditPage(){ const [items,setItems]=useState<AuditLog[]>([]); useEffect(()=>{getJson<AuditLog[]>('audit-logs').then(setItems)},[]); return <div className="space-y-4"><PageHeader title="审计日志" desc="登录、证书、域名、反代、代理入口、Runtime 操作记录。"/><DataTable headers={['时间','操作','资源','IP','User-Agent','详情']}>{items.map(x=><tr key={x.id}><td>{x.createdAt}</td><td>{x.action}</td><td>{x.resourceType}:{x.resourceId}</td><td>{x.ip}</td><td>{x.userAgent}</td><td><pre className="whitespace-pre-wrap text-xs">{x.detail}</pre></td></tr>)}</DataTable></div> }

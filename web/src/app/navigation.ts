import { Activity, Boxes, Globe, Network, ScrollText, Settings, type LucideIcon } from 'lucide-react'

export type Page = 'dashboard'|'domains'|'reverse'|'vless'|'settings'|'audit'

export const navItems: [Page,string,LucideIcon][] = [
  ['dashboard','Dashboard',Activity],
  ['domains','域名管理',Globe],
  ['reverse','反向代理',Network],
  ['vless','代理管理',Boxes],
  ['settings','系统设置',Settings],
  ['audit','审计日志',ScrollText],
]

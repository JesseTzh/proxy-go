import React from 'react'
import { Button } from '@/components/ui/button'
import { navItems, type Page } from './navigation'

export function Shell({page,setPage,children}:{page:Page; setPage:(p:Page)=>void; children:React.ReactNode}){
  return (
    <div className="flex min-h-screen flex-col md:flex-row" data-testid="app-shell">
      <aside className="shrink-0 bg-neutral-950 p-4 text-white md:min-h-screen md:w-64" data-testid="app-sidebar">
        <div className="mb-4 text-2xl font-semibold tracking-[-0.06em] md:mb-6" data-testid="app-brand">Proxy-Go</div>
        <nav className="flex gap-2 overflow-x-auto pb-1 md:grid md:overflow-visible md:pb-0" data-testid="app-nav">
          {navItems.map(([key,label,Icon])=>(
            <Button
              key={key}
              variant={page===key ? 'secondary' : 'ghost'}
              className={`min-w-max justify-start gap-3 md:w-full ${page===key ? 'bg-white text-neutral-950 hover:bg-white/90' : 'text-white hover:bg-white/10 hover:text-white'}`}
              onClick={()=>setPage(key)}
              data-testid={`nav-${key}`}
            >
              <Icon size={18} aria-hidden="true" />
              {label}
            </Button>
          ))}
        </nav>
      </aside>
      <main className="min-w-0 flex-1 overflow-auto p-4 md:p-6" data-testid="app-main">
        {children}
      </main>
    </div>
  )
}

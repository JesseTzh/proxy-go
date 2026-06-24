import { BadgeCheck } from 'lucide-react'

export function PageHeader({ title, desc, 'data-testid': dataTestId }: { title: string; desc: string; 'data-testid'?: string }) {
  return (
    <div data-testid={dataTestId}>
      <h1 className="flex items-center gap-2 text-[32px] font-semibold tracking-[-0.055em] text-[#171717] text-balance">
        <BadgeCheck size={24} aria-hidden="true" />
        {title}
      </h1>
      <p className="mt-1 max-w-2xl text-sm leading-6 text-neutral-500">{desc}</p>
    </div>
  )
}

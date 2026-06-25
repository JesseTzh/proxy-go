import { z } from 'zod'

export const inboundSchema = z.object({
  name: z.string().min(1, '请输入名称'),
  template: z.literal('vless-xhttp').default('vless-xhttp'),
  domainId: z.coerce.number().int().positive('请选择域名'),
  xhttpPath: z.string().default('/xhttp'),
  xhttpMode: z.string().default('auto'),
  realityHandshakeServer: z.string().default('www.cloudflare.com'),
  realityHandshakePort: z.coerce.number().int().min(1).max(65535).default(443),
  realityMaxTimeDiff: z.coerce.number().int().min(0).default(60),
  enabled: z.boolean(),
})

export type InboundFormValues = z.infer<typeof inboundSchema>
export type InboundFormInput = z.input<typeof inboundSchema>

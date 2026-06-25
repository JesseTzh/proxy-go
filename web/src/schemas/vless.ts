import { z } from 'zod'

export const inboundSchema = z.object({
  name: z.string().min(1, '请输入名称'),
  domainId: z.coerce.number().int().positive('请选择域名'),
  xhttpPath: z.string().default('/xhttp'),
  realityHandshakeServer: z.string().trim().min(1, '请输入 REALITY 握手服务器').default('apple.com'),
})

export type InboundFormValues = z.infer<typeof inboundSchema>
export type InboundFormInput = z.input<typeof inboundSchema>

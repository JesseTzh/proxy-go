import { z } from 'zod'

export const inboundSchema = z.object({
  template: z.enum(['vless-reality-vision', 'anytls']).default('vless-reality-vision'),
  name: z.string().min(1, '请输入名称'),
  domainId: z.coerce.number().int().positive('请选择域名'),
  realityHandshakeServer: z.string().trim().default('apple.com'),
}).superRefine((value, ctx) => {
  if (value.template === 'vless-reality-vision' && !value.realityHandshakeServer) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['realityHandshakeServer'],
      message: '请输入 REALITY 握手服务器',
    })
  }
})

export type InboundFormValues = z.infer<typeof inboundSchema>
export type InboundFormInput = z.input<typeof inboundSchema>

import { z } from 'zod'

export const inboundSchema = z.object({
  template: z.enum(['vless-reality-vision', 'anytls', 'hysteria2']).default('vless-reality-vision'),
  name: z.string().min(1, '请输入名称'),
  domainId: z.coerce.number().int().positive('请选择域名'),
  realityHandshakeServer: z.string().trim().default('apple.com'),
  hysteria2ObfsType: z.enum(['', 'salamander', 'gecko']).default('salamander'),
  hysteria2ObfsPassword: z.string().trim().default(''),
  hysteria2UpMbps: z.coerce.number().int().nonnegative().default(0),
  hysteria2DownMbps: z.coerce.number().int().nonnegative().default(0),
  hysteria2Masquerade: z.string().trim().default(''),
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

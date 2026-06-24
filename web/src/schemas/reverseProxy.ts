import { z } from 'zod'

export const reverseProxySchema = z.object({
  domainId: z.coerce.number().int().positive('请选择域名'),
  targetScheme: z.enum(['http', 'https']),
  targetHost: z.string().min(1, '请输入目标主机'),
  targetPort: z.coerce.number().int().min(1).max(65535),
  preserveHost: z.boolean(),
  webSocket: z.boolean(),
  passRealIp: z.boolean(),
  enabled: z.boolean(),
  remark: z.string(),
})

export type ReverseProxyFormValues = z.infer<typeof reverseProxySchema>
export type ReverseProxyFormInput = z.input<typeof reverseProxySchema>

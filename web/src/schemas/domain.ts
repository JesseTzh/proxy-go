import { z } from 'zod'

export const domainSchema = z.object({
  domain: z.string().min(1, '请输入域名').regex(/^[a-z0-9.-]+$/i, '域名格式不正确'),
})

export type DomainFormValues = z.infer<typeof domainSchema>

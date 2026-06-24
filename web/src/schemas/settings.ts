import { z } from 'zod'

export const settingsSchema = z.object({
  acmeEmail: z.string().email('邮箱格式不正确').or(z.literal('')),
  managementDomain: z.string(),
})

export type SettingsFormValues = z.infer<typeof settingsSchema>

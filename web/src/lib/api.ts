import ky from 'ky'

export const api = ky.create({ prefix: '/api', credentials: 'include', timeout: 15000 })
export async function getJson<T>(url: string): Promise<T> { return api.get(url).json<T>() }
export async function postJson<T>(url: string, json?: unknown): Promise<T> { return api.post(url, { json }).json<T>() }
export async function putJson<T>(url: string, json?: unknown): Promise<T> { return api.put(url, { json }).json<T>() }
export async function delJson<T>(url: string): Promise<T> { return api.delete(url).json<T>() }

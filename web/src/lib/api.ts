import ky from 'ky'
import { beginApiRequest, endApiRequest, showApiError } from './apiUi'

export const api = ky.create({ prefix: '/api', credentials: 'include', timeout: 15000 })

type ApiEnvelope<T> = {
  ok: boolean
  data?: T
  error?: {
    message?: string
  }
}

type RequestOptions = {
  loading?: boolean
  silentError?: boolean
}

type NormalizedRequestOptions = {
  loading: boolean
  silentError: boolean
}

export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export async function getJson<T>(url: string, options: RequestOptions = {}): Promise<T> {
  return requestJson<T>('get', url, undefined, normalizeOptions(options, false))
}

export async function postJson<T>(url: string, json?: unknown, options: RequestOptions = {}): Promise<T> {
  return requestJson<T>('post', url, json, normalizeOptions(options, true))
}

export async function putJson<T>(url: string, json?: unknown, options: RequestOptions = {}): Promise<T> {
  return requestJson<T>('put', url, json, normalizeOptions(options, true))
}

export async function delJson<T>(url: string, options: RequestOptions = {}): Promise<T> {
  return requestJson<T>('delete', url, undefined, normalizeOptions(options, true))
}

function normalizeOptions(options: RequestOptions, defaultLoading: boolean): NormalizedRequestOptions {
  return {
    loading: options.loading ?? defaultLoading,
    silentError: options.silentError ?? false,
  }
}

async function requestJson<T>(method: 'get' | 'post' | 'put' | 'delete', url: string, json: unknown, options: NormalizedRequestOptions): Promise<T> {
  beginApiRequest(options.loading)
  try {
    const response = await api(url, {
      method,
      json,
      throwHttpErrors: false,
    })
    const body = await parseJson(response)
    const message = errorMessage(body)

    if (!response.ok) {
      throw new ApiError(message ?? `${response.status} ${response.statusText}`, response.status)
    }

    if (isApiEnvelope<T>(body)) {
      if (!body.ok) {
        throw new ApiError(message ?? '请求失败', response.status)
      }
      return body.data as T
    }

    return body as T
  } catch (error) {
    if (!options.silentError) showApiError(readableError(error))
    throw error
  } finally {
    endApiRequest(options.loading)
  }
}

async function parseJson(response: Response): Promise<unknown> {
  const text = await response.text()
  if (!text) return undefined
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

function isApiEnvelope<T>(body: unknown): body is ApiEnvelope<T> {
  return Boolean(body && typeof body === 'object' && 'ok' in body)
}

function errorMessage(body: unknown): string | undefined {
  if (!body || typeof body !== 'object') return undefined
  const error = (body as { error?: unknown }).error
  if (typeof error === 'string') return error
  if (error && typeof error === 'object') {
    const message = (error as { message?: unknown }).message
    if (typeof message === 'string' && message) return message
  }
  return undefined
}

function readableError(error: unknown): string {
  if (error instanceof ApiError) return error.message
  if (error instanceof Error && error.message) return error.message
  return '请求失败，请稍后重试'
}

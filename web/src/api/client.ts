import type { ApiEnvelope, ApiErrorEnvelope } from './types'

const baseURL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class ApiError extends Error {
  constructor(public readonly code: string, message: string, public readonly requestId: string, public readonly fields: Record<string, string>) {
    super(message)
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${baseURL}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', 'X-Actor-ID': 'manager-local', 'X-Actor-Role': 'manager', ...(init.headers ?? {}) }
  })
  const body = await response.json() as ApiEnvelope<T> | ApiErrorEnvelope
  if (!response.ok) {
    const error = (body as ApiErrorEnvelope).error
    throw new ApiError(error.code, error.message, error.request_id, Object.fromEntries(error.field_errors.map(item => [item.field, item.message])))
  }
  return (body as ApiEnvelope<T>).data
}

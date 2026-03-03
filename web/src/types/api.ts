export type ApiErrorData = { message?: string; error?: string }

export function isApiErrorData(value: unknown): value is ApiErrorData {
  if (!value || typeof value !== 'object') {return false}
  const obj = value as Record<string, unknown>
  if ('message' in obj && typeof obj.message !== 'string') {return false}
  if ('error' in obj && typeof obj.error !== 'string') {return false}
  return true
}

export class ApiError extends Error {
  status: number
  data: ApiErrorData | null

  constructor(status: number, data: ApiErrorData | null) {
    super(data?.message ?? data?.error ?? `HTTP Error ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.data = data
  }
}

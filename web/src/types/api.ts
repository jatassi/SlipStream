export class ApiError extends Error {
  status: number
  data: { message?: string; error?: string } | null

  constructor(status: number, data: { message?: string; error?: string } | null) {
    super(data?.message ?? data?.error ?? `HTTP Error ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.data = data
  }
}


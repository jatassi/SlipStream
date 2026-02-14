export class ApiError extends Error {
  status: number
  data: { message?: string; error?: string } | null

  constructor(status: number, data: { message?: string; error?: string } | null) {
    super(data?.message || data?.error || `HTTP Error ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.data = data
  }
}

export type PaginatedResponse<T> = {
  items: T[]
  page: number
  pageSize: number
  totalCount: number
  totalPages: number
}

export type ApiResponse<T> = {
  data: T
  message?: string
}

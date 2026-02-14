import { ApiError } from '@/types'

const API_BASE = '/api/v1/requests'

// Initialize token from localStorage immediately on module load
function getInitialToken(): string | null {
  try {
    const stored = localStorage.getItem('slipstream-portal-auth')
    if (stored) {
      const parsed: unknown = JSON.parse(stored)
      if (
        parsed &&
        typeof parsed === 'object' &&
        'state' in parsed &&
        parsed.state &&
        typeof parsed.state === 'object' &&
        'token' in parsed.state &&
        typeof parsed.state.token === 'string'
      ) {
        return parsed.state.token
      }
    }
  } catch {
    // Ignore errors
  }
  return null
}

let authToken: string | null = getInitialToken()

export function setPortalAuthToken(token: string | null) {
  authToken = token
}

export function getPortalAuthToken(): string | null {
  return authToken
}

export async function portalFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  if (authToken) {
    headers.Authorization = `Bearer ${authToken}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      ...headers,
      ...options?.headers,
    },
  })

  if (!res.ok) {
    let errorData: unknown = null
    try {
      errorData = (await res.json()) as unknown
    } catch {
      // Response might not be JSON
    }
    throw new ApiError(
      res.status,
      errorData as { message?: string; error?: string } | null,
    )
  }

  if (res.status === 204) {
    return undefined as T
  }

  return (await res.json()) as T
}

export function buildQueryString(params: object): string {
  const searchParams = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      searchParams.append(key, String(value))
    }
  }
  const queryString = searchParams.toString()
  return queryString ? `?${queryString}` : ''
}

import { ApiError } from '@/types'

const API_BASE = '/api/v1/requests'

// Initialize token from localStorage immediately on module load
function getInitialToken(): string | null {
  try {
    const stored = localStorage.getItem('slipstream-portal-auth')
    if (stored) {
      const { state } = JSON.parse(stored)
      return state?.token || null
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

export async function portalFetch<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      ...headers,
      ...options?.headers,
    },
  })

  if (!res.ok) {
    let errorData = null
    try {
      errorData = await res.json()
    } catch {
      // Response might not be JSON
    }
    throw new ApiError(res.status, errorData)
  }

  if (res.status === 204) {
    return undefined as T
  }

  return res.json()
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

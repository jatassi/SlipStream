import { ApiError } from '@/types'
import { getPortalAuthToken } from './portal/client'

const API_BASE = '/api/v1'

function getInitialAdminToken(): string | null {
  try {
    const stored = localStorage.getItem('slipstream-portal-auth')
    if (stored) {
      const { state } = JSON.parse(stored)
      if (state?.user?.isAdmin && state?.token) {
        return state.token
      }
    }
  } catch {
    // Ignore parse errors
  }
  return null
}

let adminAuthToken: string | null = getInitialAdminToken()

export function setAdminAuthToken(token: string | null): void {
  adminAuthToken = token
}

export function getAdminAuthToken(): string | null {
  return adminAuthToken
}

export async function apiFetch<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  // Use admin token if available, otherwise fall back to portal token
  const token = adminAuthToken || getPortalAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      ...headers,
      ...options?.headers,
    },
  })

  if (!res.ok) {
    // Handle 401 - clear auth and redirect to login
    if (res.status === 401) {
      const hadToken = !!(adminAuthToken || getPortalAuthToken())
      adminAuthToken = null
      if (hadToken) {
        window.dispatchEvent(new CustomEvent('auth:unauthorized'))
      }
    }
    let errorData = null
    try {
      errorData = await res.json()
    } catch {
      // Response might not be JSON
    }
    throw new ApiError(res.status, errorData)
  }

  // Handle 204 No Content
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

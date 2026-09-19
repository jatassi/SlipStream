import type { APIRequestContext } from '@playwright/test'

import { adminPin, adminUsername, apiBase, authStorageKey, frontendOrigin } from './paths'

type AuthPayload = {
  token: string
  user: Record<string, unknown>
}

type StorageState = {
  cookies: []
  origins: {
    origin: string
    localStorage: { name: string; value: string }[]
  }[]
}

function readString(value: object, key: string): string {
  if (!(key in value)) {
    throw new TypeError(`missing ${key}`)
  }
  const field = value[key as keyof typeof value]
  if (typeof field !== 'string') {
    throw new TypeError(`invalid ${key}`)
  }
  return field
}

function readUser(value: object): Record<string, unknown> {
  if (!('user' in value) || typeof value.user !== 'object' || value.user === null) {
    throw new TypeError('invalid auth user')
  }
  return { ...value.user }
}

function readPayload(value: unknown): AuthPayload {
  if (!value || typeof value !== 'object') {
    throw new TypeError('invalid auth payload')
  }
  return {
    token: readString(value, 'token'),
    user: { ...readUser(value), isAdmin: true },
  }
}

function requiresSetup(value: unknown): boolean {
  return Boolean(
    value && typeof value === 'object' && 'requiresSetup' in value && value.requiresSetup,
  )
}

async function postAuth(request: APIRequestContext, path: string, body: object): Promise<AuthPayload> {
  const response = await request.post(`${apiBase}${path}`, { data: body })
  if (!response.ok()) {
    throw new Error(`${path} failed: ${response.status()}`)
  }
  return readPayload(await response.json())
}

export async function createAdminSession(request: APIRequestContext): Promise<AuthPayload> {
  const statusResponse = await request.get(`${apiBase}/auth/status`)
  if (!statusResponse.ok()) {
    throw new Error(`auth status failed: ${statusResponse.status()}`)
  }
  if (requiresSetup(await statusResponse.json())) {
    return postAuth(request, '/auth/setup', { password: adminPin })
  }
  return postAuth(request, '/requests/auth/login', {
    username: adminUsername,
    password: adminPin,
  })
}

export function storageStateFromAuth(payload: AuthPayload): StorageState {
  return {
    cookies: [],
    origins: [
      {
        origin: frontendOrigin,
        localStorage: [
          {
            name: authStorageKey,
            value: JSON.stringify({
              state: {
                token: payload.token,
                user: payload.user,
                isAuthenticated: true,
                redirectUrl: null,
              },
              version: 0,
            }),
          },
        ],
      },
    ],
  }
}

import type {
  PublicKeyCredentialCreationOptionsJSON,
  PublicKeyCredentialRequestOptionsJSON,
} from '@simplewebauthn/browser'
import { startAuthentication, startRegistration } from '@simplewebauthn/browser'

import type { PortalUser } from '@/types'

import { getPortalAuthToken, portalFetch } from './client'

const API_BASE = '/api/v1/requests'

type BeginRegistrationResponse = {
  challengeId: string
  options: {
    publicKey: PublicKeyCredentialCreationOptionsJSON
  }
}

type BeginLoginResponse = {
  challengeId: string
  options: {
    publicKey: PublicKeyCredentialRequestOptionsJSON
  }
}

type PasskeyCredential = {
  id: string
  name: string
  createdAt: string
  lastUsedAt: string | null
}

type PasskeyLoginResponse = {
  token: string
  user: PortalUser
  isAdmin: boolean
}

export const passkeyApi = {
  isSupported: () => {
    if (globalThis.window === undefined) {
      return false
    }
    return (
      globalThis.PublicKeyCredential !== undefined &&
      typeof globalThis.PublicKeyCredential === 'function'
    )
  },

  // Registration flow
  async beginRegistration(pin: string): Promise<BeginRegistrationResponse> {
    return portalFetch<BeginRegistrationResponse>('/auth/passkey/register/begin', {
      method: 'POST',
      body: JSON.stringify({ pin }),
    })
  },

  async registerPasskey(pin: string, name: string): Promise<void> {
    const { challengeId, options } = await this.beginRegistration(pin)

    const credential = await startRegistration({ optionsJSON: options.publicKey })

    const authToken = getPortalAuthToken()
    const res = await fetch(
      `${API_BASE}/auth/passkey/register/finish?challengeId=${encodeURIComponent(challengeId)}&name=${encodeURIComponent(name)}`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(authToken ? { Authorization: `Bearer ${authToken}` } : {}),
        },
        body: JSON.stringify(credential),
      },
    )

    if (!res.ok) {
      const error = await res.json().catch(() => ({ message: 'Registration failed' }))
      throw new Error(error.message || 'Registration failed')
    }
  },

  // Login flow
  async beginLogin(): Promise<BeginLoginResponse> {
    return portalFetch<BeginLoginResponse>('/auth/passkey/login/begin', {
      method: 'POST',
    })
  },

  async loginWithPasskey(): Promise<PasskeyLoginResponse> {
    const { challengeId, options } = await this.beginLogin()

    const credential = await startAuthentication({ optionsJSON: options.publicKey })

    const res = await fetch(
      `${API_BASE}/auth/passkey/login/finish?challengeId=${encodeURIComponent(challengeId)}`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credential),
      },
    )

    if (!res.ok) {
      const error = await res.json().catch(() => ({ message: 'Login failed' }))
      throw new Error(error.message || 'Login failed')
    }

    return res.json()
  },

  // Credential management
  listCredentials: () => portalFetch<PasskeyCredential[]>('/auth/passkey/credentials'),

  updateCredential: (id: string, name: string) =>
    portalFetch<void>(`/auth/passkey/credentials/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ name }),
    }),

  deleteCredential: (id: string) =>
    portalFetch<void>(`/auth/passkey/credentials/${id}`, {
      method: 'DELETE',
    }),
}

export type { PasskeyCredential, PasskeyLoginResponse }

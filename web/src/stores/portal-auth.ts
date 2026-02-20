import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { setAdminAuthToken } from '@/api/client'
import { setPortalAuthToken } from '@/api/portal/client'
import type { PortalUser } from '@/types'

type PortalAuthState = {
  token: string | null
  user: PortalUser | null
  redirectUrl: string | null
  isAuthenticated: boolean

  login: (token: string, user: PortalUser) => void
  logout: () => void
  setUser: (user: PortalUser) => void
  setRedirectUrl: (url: string | null) => void
  getPostLoginRedirect: () => string
}

export const usePortalAuthStore = create<PortalAuthState>()(
  persist(
    (set, get) => ({
      token: null,
      user: null,
      redirectUrl: null,
      isAuthenticated: false,

      login: (token, user) => {
        // Set token in portal API client
        setPortalAuthToken(token)
        // If admin, also set token in main API client
        if (user.isAdmin) {
          setAdminAuthToken(token)
        }
        set({ token, user, isAuthenticated: true })
      },

      logout: () => {
        setPortalAuthToken(null)
        setAdminAuthToken(null)
        set({ token: null, user: null, isAuthenticated: false })
      },

      setUser: (user) => {
        set({ user })
      },

      setRedirectUrl: (url) => {
        set({ redirectUrl: url })
      },

      getPostLoginRedirect: () => {
        const state = get()
        // Admin users redirect to saved URL or dashboard
        if (state.user?.isAdmin) {
          const redirect = state.redirectUrl
          set({ redirectUrl: null })
          if (redirect && redirect.startsWith('/') && !redirect.startsWith('/requests/auth/') && redirect !== '/auth/setup') {
            return redirect
          }
          return '/'
        }
        // Portal users always go to requests page
        return '/requests'
      },
    }),
    {
      name: 'slipstream-portal-auth',
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        redirectUrl: state.redirectUrl,
      }),
    },
  ),
)

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import * as authApi from '@/api/auth'
import { usePortalAuthStore } from '@/stores/portalAuth'

export const adminAuthKeys = {
  all: ['adminAuth'] as const,
  status: () => [...adminAuthKeys.all, 'status'] as const,
}

export function useAuthStatus() {
  return useQuery({
    queryKey: adminAuthKeys.status(),
    queryFn: () => authApi.getAuthStatus(),
    staleTime: 30_000,
    retry: false,
  })
}

export function useAdminSetup() {
  const queryClient = useQueryClient()
  const { login: storeLogin } = usePortalAuthStore()

  return useMutation({
    mutationFn: (password: string) => authApi.adminSetup(password),
    onSuccess: (response) => {
      // Login as admin with the returned token
      storeLogin(response.token, { ...response.user, isAdmin: true })
      queryClient.invalidateQueries({ queryKey: adminAuthKeys.status() })
    },
  })
}

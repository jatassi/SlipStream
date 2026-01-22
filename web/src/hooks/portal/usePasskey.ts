import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { passkeyApi } from '@/api/portal/passkey'
import { usePortalAuthStore } from '@/stores/portalAuth'
import { toast } from 'sonner'

export const passkeyKeys = {
  all: ['passkey'] as const,
  credentials: () => [...passkeyKeys.all, 'credentials'] as const,
}

export function usePasskeySupport() {
  // Check support synchronously - this is a sync browser API check
  const isSupported = passkeyApi.isSupported()
  return { isSupported, isLoading: false }
}

export function usePasskeyCredentials() {
  const { isAuthenticated } = usePortalAuthStore()

  return useQuery({
    queryKey: passkeyKeys.credentials(),
    queryFn: () => passkeyApi.listCredentials(),
    enabled: isAuthenticated,
  })
}

export function useRegisterPasskey() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ pin, name }: { pin: string; name: string }) =>
      passkeyApi.registerPasskey(pin, name),
    onSuccess: async () => {
      await queryClient.refetchQueries({ queryKey: passkeyKeys.credentials() })
      toast.success('Passkey registered successfully')
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to register passkey')
    },
  })
}

export function useDeletePasskey() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => passkeyApi.deleteCredential(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: passkeyKeys.credentials() })
      toast.success('Passkey deleted')
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to delete passkey')
    },
  })
}

export function useUpdatePasskeyName() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      passkeyApi.updateCredential(id, name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: passkeyKeys.credentials() })
      toast.success('Passkey renamed')
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to rename passkey')
    },
  })
}

export function usePasskeyLogin() {
  const { login: storeLogin } = usePortalAuthStore()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => passkeyApi.loginWithPasskey(),
    onSuccess: (data) => {
      storeLogin(data.token, data.user)
      queryClient.invalidateQueries()
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Passkey login failed')
    },
  })
}

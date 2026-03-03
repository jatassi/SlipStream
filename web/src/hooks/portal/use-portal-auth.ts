import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { portalAuthApi } from '@/api'
import { portalLibraryKeys } from '@/hooks/portal/use-portal-library'
import { requestKeys } from '@/hooks/portal/use-requests'
import { userNotificationKeys } from '@/hooks/portal/use-user-notifications'
import { createQueryKeys } from '@/lib/query-keys'
import { usePortalAuthStore } from '@/stores/portal-auth'
import type { LoginRequest, SignupRequest, UpdateProfileRequest } from '@/types'

const baseKeys = createQueryKeys('portalAuth')
const portalAuthKeys = {
  ...baseKeys,
  profile: () => [...baseKeys.all, 'profile'] as const,
}

const baseInvitationKeys = createQueryKeys('invitation')
const invitationKeys = {
  ...baseInvitationKeys,
  validate: (token: string) => [...baseInvitationKeys.all, token] as const,
}

export function usePortalLogin() {
  const queryClient = useQueryClient()
  const { login: storeLogin } = usePortalAuthStore()

  return useMutation({
    mutationFn: (data: LoginRequest) => portalAuthApi.login(data),
    onSuccess: (response) => {
      storeLogin(response.token, response.user)
      void queryClient.invalidateQueries({ queryKey: portalAuthKeys.all })
    },
  })
}

export function usePortalSignup() {
  const queryClient = useQueryClient()
  const { login: storeLogin } = usePortalAuthStore()

  return useMutation({
    mutationFn: (data: SignupRequest) => portalAuthApi.signup(data),
    onSuccess: (response) => {
      storeLogin(response.token, response.user)
      void queryClient.invalidateQueries({ queryKey: portalAuthKeys.all })
    },
  })
}

export function usePortalLogout() {
  const queryClient = useQueryClient()
  const { logout: storeLogout } = usePortalAuthStore()

  return useMutation({
    mutationFn: () => portalAuthApi.logout(),
    onSuccess: () => {
      storeLogout()
      queryClient.removeQueries({ queryKey: portalAuthKeys.all })
      queryClient.removeQueries({ queryKey: requestKeys.all })
      queryClient.removeQueries({ queryKey: portalLibraryKeys.all })
      queryClient.removeQueries({ queryKey: userNotificationKeys.all })
      queryClient.removeQueries({ queryKey: invitationKeys.all })
    },
    onError: () => {
      storeLogout()
      queryClient.removeQueries({ queryKey: portalAuthKeys.all })
      queryClient.removeQueries({ queryKey: requestKeys.all })
      queryClient.removeQueries({ queryKey: portalLibraryKeys.all })
      queryClient.removeQueries({ queryKey: userNotificationKeys.all })
      queryClient.removeQueries({ queryKey: invitationKeys.all })
    },
  })
}

export function useUpdatePortalProfile() {
  const queryClient = useQueryClient()
  const { setUser } = usePortalAuthStore()

  return useMutation({
    mutationFn: (data: UpdateProfileRequest) => portalAuthApi.updateProfile(data),
    onSuccess: (user) => {
      setUser(user)
      queryClient.setQueryData(portalAuthKeys.profile(), user)
    },
  })
}

export function useValidateInvitation(token: string) {
  return useQuery({
    queryKey: invitationKeys.validate(token),
    queryFn: () => portalAuthApi.validateInvitation(token),
    enabled: !!token,
    retry: false,
  })
}

export function useVerifyPin() {
  return useMutation({
    mutationFn: (pin: string) => portalAuthApi.verifyPin(pin),
  })
}

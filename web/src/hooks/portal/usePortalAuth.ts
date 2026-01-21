import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { portalAuthApi } from '@/api'
import { usePortalAuthStore } from '@/stores/portalAuth'
import type { LoginRequest, SignupRequest, UpdateProfileRequest } from '@/types'

export const portalAuthKeys = {
  all: ['portalAuth'] as const,
  profile: () => [...portalAuthKeys.all, 'profile'] as const,
}

export function usePortalProfile() {
  const { isAuthenticated } = usePortalAuthStore()

  return useQuery({
    queryKey: portalAuthKeys.profile(),
    queryFn: () => portalAuthApi.getProfile(),
    enabled: isAuthenticated,
  })
}

export function usePortalLogin() {
  const queryClient = useQueryClient()
  const { login: storeLogin } = usePortalAuthStore()

  return useMutation({
    mutationFn: (data: LoginRequest) => portalAuthApi.login(data),
    onSuccess: (response) => {
      storeLogin(response.token, response.user)
      queryClient.invalidateQueries({ queryKey: portalAuthKeys.all })
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
      queryClient.invalidateQueries({ queryKey: portalAuthKeys.all })
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
      queryClient.clear()
    },
    onError: () => {
      storeLogout()
      queryClient.clear()
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
    queryKey: ['invitation', token] as const,
    queryFn: () => portalAuthApi.validateInvitation(token),
    enabled: !!token,
    retry: false,
  })
}

export function useResendInvitation() {
  return useMutation({
    mutationFn: (email: string) => portalAuthApi.resendInvitation(email),
  })
}

export function useVerifyPin() {
  return useMutation({
    mutationFn: (pin: string) => portalAuthApi.verifyPin(pin),
  })
}

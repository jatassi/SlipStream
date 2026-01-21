import { portalFetch } from './client'
import type {
  LoginRequest,
  LoginResponse,
  SignupRequest,
  SignupResponse,
  PortalUser,
  UpdateProfileRequest,
  ValidateInvitationResponse,
  VerifyPinResponse,
} from '@/types'

export const portalAuthApi = {
  login: (data: LoginRequest) =>
    portalFetch<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  signup: (data: SignupRequest) =>
    portalFetch<SignupResponse>('/auth/signup', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  validateInvitation: (token: string) =>
    portalFetch<ValidateInvitationResponse>(`/auth/validate-invitation?token=${encodeURIComponent(token)}`),

  resendInvitation: (username: string) =>
    portalFetch<void>('/auth/resend', {
      method: 'POST',
      body: JSON.stringify({ username }),
    }),

  getProfile: () =>
    portalFetch<PortalUser>('/auth/profile'),

  updateProfile: (data: UpdateProfileRequest) =>
    portalFetch<PortalUser>('/auth/profile', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  verifyPin: (pin: string) =>
    portalFetch<VerifyPinResponse>('/auth/verify-pin', {
      method: 'POST',
      body: JSON.stringify({ pin }),
    }),

  logout: () =>
    portalFetch<void>('/auth/logout', { method: 'POST' }),
}

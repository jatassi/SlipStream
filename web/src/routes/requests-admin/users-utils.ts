import type { Invitation, PortalUserWithQuota } from '@/types'

export function getInvitationStatus(invitation: Invitation): string {
  if (invitation.usedAt) {
    return 'Used'
  }
  if (new Date(invitation.expiresAt) < new Date()) {
    return 'Expired'
  }
  return 'Pending'
}

export function getQuotaDisplay(user: PortalUserWithQuota): string {
  if (!user.quota) {
    return 'Not set'
  }
  const parts: string[] = []
  for (const mod of user.quota.modules) {
    if (mod.quotaLimit > 0) {
      parts.push(`${mod.quotaUsed}/${mod.quotaLimit} ${mod.moduleType}`)
    }
  }
  return parts.length > 0 ? parts.join(', ') : 'Unlimited'
}

export function getProfileName(
  moduleType: string,
  moduleSettings: { moduleType: string; qualityProfileId: number | null }[],
  qualityProfiles: { id: number; name: string }[] | undefined,
): string {
  const setting = moduleSettings.find((s) => s.moduleType === moduleType)
  if (!setting?.qualityProfileId) {
    return 'Default'
  }
  const profile = qualityProfiles?.find((p) => p.id === setting.qualityProfileId)
  return profile?.name ?? 'Unknown'
}

export function userSubtitle(
  user: PortalUserWithQuota,
  qualityProfiles: { id: number; name: string }[] | undefined,
): string {
  const parts = [
    `Movie: ${getProfileName('movie', user.moduleSettings, qualityProfiles)}`,
    `TV: ${getProfileName('tv', user.moduleSettings, qualityProfiles)}`,
    `Quota: ${getQuotaDisplay(user)}`,
  ]
  if (user.autoApprove) {
    parts.push('Auto-approve')
  }
  return parts.join(' · ')
}

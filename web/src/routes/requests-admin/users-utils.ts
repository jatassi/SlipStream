import type { Invitation, PortalUserWithQuota } from '@/types'

type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline'

export function getInvitationStatus(invitation: Invitation): {
  label: string
  variant: BadgeVariant
} {
  if (invitation.usedAt) {
    return { label: 'Used', variant: 'secondary' }
  }
  if (new Date(invitation.expiresAt) < new Date()) {
    return { label: 'Expired', variant: 'destructive' }
  }
  return { label: 'Pending', variant: 'default' }
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

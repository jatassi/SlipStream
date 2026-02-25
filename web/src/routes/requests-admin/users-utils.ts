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
  if (user.quota.moviesLimit !== null) {
    parts.push(`${user.quota.moviesUsed}/${user.quota.moviesLimit} movies`)
  }
  if (user.quota.seasonsLimit !== null) {
    parts.push(`${user.quota.seasonsUsed}/${user.quota.seasonsLimit} seasons`)
  }
  if (user.quota.episodesLimit !== null) {
    parts.push(`${user.quota.episodesUsed}/${user.quota.episodesLimit} episodes`)
  }
  return parts.length > 0 ? parts.join(', ') : 'Unlimited'
}

export function getProfileName(
  profileId: number | null,
  qualityProfiles: { id: number; name: string }[] | undefined,
): string {
  if (!profileId) {
    return 'Default'
  }
  const profile = qualityProfiles?.find((p) => p.id === profileId)
  return profile?.name ?? 'Unknown'
}

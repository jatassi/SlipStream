import { UserPlus, Users } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import type { Invitation, PortalUserWithQuota } from '@/types'

import { InvitationCard } from './invitation-card'
import { UserCard } from './user-card'

type UsersTabContentProps = {
  users: PortalUserWithQuota[] | undefined
  qualityProfiles: { id: number; name: string }[] | undefined
  togglePending: boolean
  onToggleEnabled: (user: PortalUserWithQuota) => void
  onEdit: (user: PortalUserWithQuota) => void
  onDelete: (id: number) => void
  onInvite: () => void
}

export function UsersTabContent({
  users,
  qualityProfiles,
  togglePending,
  onToggleEnabled,
  onEdit,
  onDelete,
  onInvite,
}: UsersTabContentProps) {
  if (!users?.length) {
    return (
      <EmptyState
        icon={<Users className="size-8" />}
        title="No users yet"
        description="Invite users to start using the request portal"
        action={{ label: 'Invite User', onClick: onInvite }}
      />
    )
  }

  return (
    <div className="space-y-4">
      {users.map((user) => (
        <UserCard
          key={user.id}
          user={user}
          qualityProfiles={qualityProfiles}
          togglePending={togglePending}
          onToggleEnabled={onToggleEnabled}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      ))}
    </div>
  )
}

type InvitationsTabContentProps = {
  invitations: Invitation[] | undefined
  copiedToken: string | null
  expandedLinkToken: string | null
  resendPending: boolean
  onCopyLink: (token: string) => void
  onToggleLink: (token: string) => void
  onResend: (id: number) => void
  onDelete: (id: number) => void
  onInvite: () => void
}

export function InvitationsTabContent({
  invitations,
  copiedToken,
  expandedLinkToken,
  resendPending,
  onCopyLink,
  onToggleLink,
  onResend,
  onDelete,
  onInvite,
}: InvitationsTabContentProps) {
  if (!invitations?.length) {
    return (
      <EmptyState
        icon={<UserPlus className="size-8" />}
        title="No invitations"
        description="Create an invitation to add new users"
        action={{ label: 'Create Invitation', onClick: onInvite }}
      />
    )
  }

  return (
    <div className="space-y-4">
      {invitations.map((invitation) => (
        <InvitationCard
          key={invitation.id}
          invitation={invitation}
          copiedToken={copiedToken}
          expandedLinkToken={expandedLinkToken}
          resendPending={resendPending}
          onCopyLink={onCopyLink}
          onToggleLink={onToggleLink}
          onResend={onResend}
          onDelete={onDelete}
        />
      ))}
    </div>
  )
}

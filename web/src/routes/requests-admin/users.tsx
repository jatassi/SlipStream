import { formatDistanceToNow } from 'date-fns'
import { AlertTriangle, UserPlus, Users } from 'lucide-react'

import { Group, IconTile, Row } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import type { SettingsRowAction } from '@/components/settings/settings-item-row'
import { SettingsItemRow } from '@/components/settings/settings-item-row'
import { AddAction, SettingsList } from '@/components/settings/settings-list'
import type { Invitation, PortalUserWithQuota } from '@/types'

import { InviteSheet } from './invite-sheet'
import { useRequestUsersPage } from './use-request-users-page'
import { UserEditSheet } from './user-edit-sheet'
import { getInvitationStatus, userSubtitle } from './users-utils'

type PageState = ReturnType<typeof useRequestUsersPage>

function PortalDisabledGroup({ enabled }: { enabled: boolean }) {
  if (enabled) {
    return null
  }
  return (
    <Group>
      <Row
        tone="warning"
        leading={
          <IconTile className="bg-amber-500">
            <AlertTriangle />
          </IconTile>
        }
        title="The requests portal is disabled"
        subtitle="Portal users cannot sign in or submit requests"
        href="/requests-admin/settings"
        chevron
      />
    </Group>
  )
}

function UserRow({ user, page }: { user: PortalUserWithQuota; page: PageState }) {
  const actions: SettingsRowAction[] = [
    {
      label: 'Delete user',
      destructive: true,
      onClick: () => page.handleDeleteUser(user.id),
      confirm: {
        title: `Delete ${user.username}?`,
        description: 'Their requests are preserved. This cannot be undone.',
      },
    },
  ]

  return (
    <SettingsItemRow
      leading={
        <IconTile className="bg-violet-600">
          <Users />
        </IconTile>
      }
      title={user.username}
      subtitle={userSubtitle(user, page.qualityProfiles)}
      onOpen={() => {
        page.handleOpenEdit(user)
      }}
      openLabel={`Edit ${user.username}`}
      toggle={{
        label: `${user.username} enabled`,
        checked: user.enabled,
        onCheckedChange: () => void page.handleToggleEnabled(user),
        disabled: page.togglePending,
      }}
      actions={actions}
    />
  )
}

function invitationSubtitle(invitation: Invitation): string {
  const expiry = invitation.usedAt
    ? []
    : [`expires ${formatDistanceToNow(new Date(invitation.expiresAt), { addSuffix: true })}`]
  return [
    getInvitationStatus(invitation),
    `created ${formatDistanceToNow(new Date(invitation.createdAt), { addSuffix: true })}`,
    ...expiry,
  ].join(' · ')
}

function invitationActions(invitation: Invitation, page: PageState): SettingsRowAction[] {
  const unused: SettingsRowAction[] = invitation.usedAt
    ? []
    : [
        {
          label: 'Copy link',
          onClick: () => {
            page.handleCopyLink(invitation.token)
          },
        },
        {
          label: 'Resend',
          onClick: () => page.handleResendInvitation(invitation.id),
        },
      ]

  return [
    ...unused,
    {
      label: 'Delete invitation',
      destructive: true,
      onClick: () => page.handleDeleteInvitation(invitation.id),
      confirm: {
        title: `Delete the invitation for ${invitation.username}?`,
        description: 'The link stops working immediately.',
      },
    },
  ]
}

function InvitationRow({ invitation, page }: { invitation: Invitation; page: PageState }) {
  return (
    <SettingsItemRow
      leading={
        <IconTile className="bg-sky-600">
          <UserPlus />
        </IconTile>
      }
      title={invitation.username}
      subtitle={invitationSubtitle(invitation)}
      actions={invitationActions(invitation, page)}
    />
  )
}

function UsersGroup({ page }: { page: PageState }) {
  return (
    <SettingsList state={page.usersState} header="Users" empty="No users yet">
      {page.users?.map((user) => <UserRow key={user.id} user={user} page={page} />)}
    </SettingsList>
  )
}

function InvitationsGroup({ page }: { page: PageState }) {
  return (
    <SettingsList state={page.invitationsState} header="Invitations" empty="No invitations">
      {page.invitations?.map((invitation) => (
        <InvitationRow key={invitation.id} invitation={invitation} page={page} />
      ))}
    </SettingsList>
  )
}

function UserSheets({ page }: { page: PageState }) {
  return (
    <>
      <InviteSheet
        open={page.showInviteSheet}
        onOpenChange={page.setShowInviteSheet}
        inviteName={page.inviteName}
        onNameChange={page.setInviteName}
        moduleSettings={page.inviteModuleSettings}
        onModuleProfileChange={page.setInviteModuleProfile}
        autoApprove={page.inviteAutoApprove}
        onAutoApproveChange={page.setInviteAutoApprove}
        qualityProfiles={page.qualityProfiles}
        isPending={page.createMutation.isPending}
        onSubmit={() => void page.handleCreateInvitation()}
      />
      {page.editingUser === null ? null : (
        <UserEditSheet
          key={page.editingUser.id}
          user={page.editingUser}
          open={page.showUserSheet}
          onOpenChange={page.setShowUserSheet}
          qualityProfiles={page.qualityProfiles ?? []}
        />
      )}
    </>
  )
}

export function RequestUsersPage() {
  const page = useRequestUsersPage()
  const back = usePushBack()

  return (
    <Screen
      title="Users"
      back={back}
      trailing={<AddAction label="Invite user" onClick={page.handleOpenInvite} />}
    >
      <PortalDisabledGroup enabled={page.portalEnabled} />
      <UsersGroup page={page} />
      <InvitationsGroup page={page} />
      <UserSheets page={page} />
    </Screen>
  )
}

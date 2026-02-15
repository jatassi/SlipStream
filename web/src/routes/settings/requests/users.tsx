import { AlertCircle, UserPlus } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { PageHeader } from '@/components/layout/page-header'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { InviteDialog } from './invite-dialog'
import { RequestsNav } from './requests-nav'
import { useRequestUsersPage } from './use-request-users-page'
import { UserEditDialog } from './user-edit-dialog'
import { InvitationsTabContent, UsersTabContent } from './users-tab-content'

export function RequestUsersPage() {
  const state = useRequestUsersPage()

  if (state.isLoading) {
    return (
      <div>
        <PageHeader title="Portal Users" />
        <LoadingState variant="list" count={3} />
      </div>
    )
  }

  if (state.isError) {
    return (
      <div>
        <PageHeader title="Portal Users" />
        <ErrorState onRetry={state.refetch} />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="External Requests"
        description="Manage portal users and content requests"
        breadcrumbs={[
          { label: 'Settings', href: '/settings/media' },
          { label: 'External Requests' },
        ]}
        actions={
          <Button onClick={state.handleOpenInvite}>
            <UserPlus className="mr-2 size-4" />
            Invite User
          </Button>
        }
      />

      <RequestsNav />
      <PortalDisabledAlert enabled={state.portalEnabled} />
      <UsersTabs state={state} />
      <PageDialogs state={state} />
    </div>
  )
}

function PortalDisabledAlert({ enabled }: { enabled: boolean }) {
  if (enabled) {
    return null
  }

  return (
    <Alert>
      <AlertCircle className="size-4" />
      <AlertDescription>
        The external requests portal is currently disabled. Portal users cannot submit new requests
        or access the portal. You can re-enable it in the{' '}
        <a href="/settings/requests/settings" className="font-medium underline">
          Settings
        </a>{' '}
        tab.
      </AlertDescription>
    </Alert>
  )
}

function PageDialogs({ state }: { state: ReturnType<typeof useRequestUsersPage> }) {
  return (
    <>
      <InviteDialog
        open={state.showInviteDialog}
        onOpenChange={state.setShowInviteDialog}
        inviteName={state.inviteName}
        onNameChange={state.setInviteName}
        qualityProfileId={state.inviteQualityProfileId}
        onQualityProfileChange={state.setInviteQualityProfileId}
        autoApprove={state.inviteAutoApprove}
        onAutoApproveChange={state.setInviteAutoApprove}
        qualityProfiles={state.qualityProfiles}
        isPending={state.createMutation.isPending}
        onSubmit={state.handleCreateInvitation}
      />

      {state.editingUser ? (
        <UserEditDialog
          user={state.editingUser}
          open={state.showUserDialog}
          onOpenChange={state.setShowUserDialog}
          qualityProfiles={state.qualityProfiles ?? []}
        />
      ) : null}
    </>
  )
}

function UsersTabs({ state }: { state: ReturnType<typeof useRequestUsersPage> }) {
  return (
    <Tabs value={state.activeTab} onValueChange={state.setActiveTab}>
      <TabsList>
        <TabsTrigger value="users">
          Users{' '}
          {state.userCount > 0 ? (
            <Badge variant="secondary" className="ml-1">
              {state.userCount}
            </Badge>
          ) : null}
        </TabsTrigger>
        <TabsTrigger value="invitations">
          Invitations{' '}
          {state.pendingInvitationCount > 0 ? (
            <Badge variant="secondary" className="ml-1">
              {state.pendingInvitationCount}
            </Badge>
          ) : null}
        </TabsTrigger>
      </TabsList>

      <TabsContent value="users" className="mt-4">
        <UsersTabContent
          users={state.users}
          qualityProfiles={state.qualityProfiles}
          togglePending={state.togglePending}
          onToggleEnabled={state.handleToggleEnabled}
          onEdit={state.handleOpenEdit}
          onDelete={state.handleDeleteUser}
          onInvite={state.handleOpenInvite}
        />
      </TabsContent>

      <TabsContent value="invitations" className="mt-4">
        <InvitationsTabContent
          invitations={state.invitations}
          copiedToken={state.copiedToken}
          expandedLinkToken={state.expandedLinkToken}
          resendPending={state.resendMutation.isPending}
          onCopyLink={state.handleCopyLink}
          onToggleLink={state.toggleLinkVisibility}
          onResend={state.handleResendInvitation}
          onDelete={state.handleDeleteInvitation}
          onInvite={state.handleOpenInvite}
        />
      </TabsContent>
    </Tabs>
  )
}

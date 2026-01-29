import { useState } from 'react'
import { Edit, Trash2, Users, UserPlus, Copy, Check, RefreshCw, Loader2, AlertCircle, Link, ChevronUp } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { RequestsNav } from './RequestsNav'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import {
  useAdminUsers,
  useUpdateAdminUser,
  useEnableUser,
  useDisableUser,
  useDeleteAdminUser,
  useAdminInvitations,
  useCreateInvitation,
  useDeleteInvitation,
  useAdminResendInvitation,
  getInvitationLink,
  useQualityProfiles,
  usePortalEnabled,
} from '@/hooks'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { toast } from 'sonner'
import type { PortalUserWithQuota, Invitation, AdminUpdateUserInput } from '@/types'
import { formatDistanceToNow } from 'date-fns'

export function RequestUsersPage() {
  const [activeTab, setActiveTab] = useState<string>('users')
  const [showUserDialog, setShowUserDialog] = useState(false)
  const [showInviteDialog, setShowInviteDialog] = useState(false)
  const [editingUser, setEditingUser] = useState<PortalUserWithQuota | null>(null)
  const [inviteName, setInviteName] = useState('')
  const [inviteQualityProfileId, setInviteQualityProfileId] = useState<number | null>(null)
  const [inviteAutoApprove, setInviteAutoApprove] = useState(false)
  const [copiedToken, setCopiedToken] = useState<string | null>(null)
  const [expandedLinkToken, setExpandedLinkToken] = useState<string | null>(null)

  const { data: users, isLoading: usersLoading, isError: usersError, refetch: refetchUsers } = useAdminUsers()
  const { data: invitations, isLoading: invitationsLoading, isError: invitationsError, refetch: refetchInvitations } = useAdminInvitations()
  const { data: qualityProfiles } = useQualityProfiles()
  const portalEnabled = usePortalEnabled()

  const enableMutation = useEnableUser()
  const disableMutation = useDisableUser()
  const deleteMutation = useDeleteAdminUser()
  const createInvitationMutation = useCreateInvitation()
  const deleteInvitationMutation = useDeleteInvitation()
  const resendInvitationMutation = useAdminResendInvitation()

  const handleToggleEnabled = async (user: PortalUserWithQuota) => {
    try {
      if (user.enabled) {
        await disableMutation.mutateAsync(user.id)
        toast.success('User disabled')
      } else {
        await enableMutation.mutateAsync(user.id)
        toast.success('User enabled')
      }
    } catch {
      toast.error('Failed to update user')
    }
  }

  const handleOpenEdit = (user: PortalUserWithQuota) => {
    setEditingUser(user)
    setShowUserDialog(true)
  }

  const handleDeleteUser = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
      toast.success('User deleted')
    } catch {
      toast.error('Failed to delete user')
    }
  }

  const handleOpenInvite = () => {
    setInviteName('')
    setInviteQualityProfileId(null)
    setInviteAutoApprove(false)
    setShowInviteDialog(true)
  }

  const handleCreateInvitation = async () => {
    if (!inviteName.trim()) {
      toast.error('Name is required')
      return
    }
    try {
      const invitation = await createInvitationMutation.mutateAsync({
        username: inviteName,
        qualityProfileId: inviteQualityProfileId,
        autoApprove: inviteAutoApprove,
      })
      toast.success('Invitation created')
      setShowInviteDialog(false)
      setInviteName('')
      setInviteQualityProfileId(null)
      setInviteAutoApprove(false)
      void handleCopyLink(invitation.token)
    } catch (error) {
      toast.error('Failed to create invitation', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    }
  }

  const handleDeleteInvitation = async (id: number) => {
    try {
      await deleteInvitationMutation.mutateAsync(id)
      toast.success('Invitation deleted')
    } catch {
      toast.error('Failed to delete invitation')
    }
  }

  const handleResendInvitation = async (id: number) => {
    try {
      const invitation = await resendInvitationMutation.mutateAsync(id)
      toast.success('Invitation resent')
      void handleCopyLink(invitation.token)
    } catch {
      toast.error('Failed to resend invitation')
    }
  }

  const handleCopyLink = async (token: string) => {
    const link = getInvitationLink(token)
    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(link)
        setCopiedToken(token)
        toast.success('Invitation link copied to clipboard')
        setTimeout(() => setCopiedToken(null), 3000)
        return
      } catch {
        // Fall through to show link
      }
    }
    // Clipboard not available - show the link for manual copying
    setExpandedLinkToken(token)
    toast.info('Select and copy the link below')
  }

  const toggleLinkVisibility = (token: string) => {
    setExpandedLinkToken(expandedLinkToken === token ? null : token)
  }

  const getInvitationStatus = (invitation: Invitation): { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' } => {
    if (invitation.usedAt) {
      return { label: 'Used', variant: 'secondary' }
    }
    if (new Date(invitation.expiresAt) < new Date()) {
      return { label: 'Expired', variant: 'destructive' }
    }
    return { label: 'Pending', variant: 'default' }
  }

  const getQuotaDisplay = (user: PortalUserWithQuota): string => {
    if (!user.quota) return 'Not set'
    const parts = []
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

  const getProfileName = (profileId: number | null): string => {
    if (!profileId) return 'Default'
    const profile = qualityProfiles?.find((p) => p.id === profileId)
    return profile?.name || 'Unknown'
  }

  const isLoading = activeTab === 'users' ? usersLoading : invitationsLoading
  const isError = activeTab === 'users' ? usersError : invitationsError
  const refetch = activeTab === 'users' ? refetchUsers : refetchInvitations

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Portal Users" />
        <LoadingState variant="list" count={3} />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Portal Users" />
        <ErrorState onRetry={refetch} />
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
          <Button onClick={handleOpenInvite}>
            <UserPlus className="size-4 mr-2" />
            Invite User
          </Button>
        }
      />

      <RequestsNav />

      {!portalEnabled && (
        <Alert>
          <AlertCircle className="size-4" />
          <AlertDescription>
            The external requests portal is currently disabled. Portal users cannot submit new requests or access the portal.
            You can re-enable it in the <a href="/settings/requests/settings" className="underline font-medium">Settings</a> tab.
          </AlertDescription>
        </Alert>
      )}

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="users">
            Users {users && users.length > 0 && <Badge variant="secondary" className="ml-1">{users.length}</Badge>}
          </TabsTrigger>
          <TabsTrigger value="invitations">
            Invitations {invitations && invitations.filter((i) => !i.usedAt).length > 0 && (
              <Badge variant="secondary" className="ml-1">{invitations.filter((i) => !i.usedAt).length}</Badge>
            )}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="users" className="mt-4">
          {!users?.length ? (
            <EmptyState
              icon={<Users className="size-8" />}
              title="No users yet"
              description="Invite users to start using the request portal"
              action={{ label: 'Invite User', onClick: handleOpenInvite }}
            />
          ) : (
            <div className="space-y-4">
              {users.map((user) => (
                <Card key={user.id}>
                  <CardHeader className="flex flex-row items-center justify-between py-4">
                    <div className="flex items-center gap-4">
                      <div className="flex size-10 items-center justify-center rounded-full bg-muted">
                        <Users className="size-5" />
                      </div>
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          <CardTitle className="text-base">{user.displayName || user.username}</CardTitle>
                          {user.autoApprove && <Badge>Auto-Approve</Badge>}
                          {!user.enabled && <Badge variant="destructive">Disabled</Badge>}
                        </div>
                        <CardDescription className="text-xs">
                          {user.username} • Profile: {getProfileName(user.qualityProfileId)} • Quota: {getQuotaDisplay(user)}
                        </CardDescription>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <Switch
                        checked={user.enabled}
                        onCheckedChange={() => handleToggleEnabled(user)}
                        disabled={enableMutation.isPending || disableMutation.isPending}
                      />
                      <Button variant="ghost" size="icon" onClick={() => handleOpenEdit(user)}>
                        <Edit className="size-4" />
                      </Button>
                      <ConfirmDialog
                        trigger={
                          <Button variant="ghost" size="icon">
                            <Trash2 className="size-4" />
                          </Button>
                        }
                        title="Delete user"
                        description={`Are you sure you want to delete "${user.displayName || user.username}"? Their requests will be preserved.`}
                        confirmLabel="Delete"
                        variant="destructive"
                        onConfirm={() => handleDeleteUser(user.id)}
                      />
                    </div>
                  </CardHeader>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>

        <TabsContent value="invitations" className="mt-4">
          {!invitations?.length ? (
            <EmptyState
              icon={<UserPlus className="size-8" />}
              title="No invitations"
              description="Create an invitation to add new users"
              action={{ label: 'Create Invitation', onClick: handleOpenInvite }}
            />
          ) : (
            <div className="space-y-4">
              {invitations.map((invitation) => {
                const status = getInvitationStatus(invitation)
                const isLinkExpanded = expandedLinkToken === invitation.token
                return (
                  <Card key={invitation.id}>
                    <CardHeader className="flex flex-row items-center justify-between py-4">
                      <div className="flex items-center gap-4">
                        <div className="flex size-10 items-center justify-center rounded-lg bg-muted">
                          <UserPlus className="size-5" />
                        </div>
                        <div className="space-y-1">
                          <div className="flex items-center gap-2">
                            <CardTitle className="text-base">{invitation.username}</CardTitle>
                            <Badge variant={status.variant}>{status.label}</Badge>
                          </div>
                          <CardDescription className="text-xs">
                            Created {formatDistanceToNow(new Date(invitation.createdAt), { addSuffix: true })}
                            {!invitation.usedAt && ` • Expires ${formatDistanceToNow(new Date(invitation.expiresAt), { addSuffix: true })}`}
                          </CardDescription>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        {!invitation.usedAt && (
                          <>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleCopyLink(invitation.token)}
                            >
                              {copiedToken === invitation.token ? (
                                <Check className="size-4 mr-1" />
                              ) : (
                                <Copy className="size-4 mr-1" />
                              )}
                              Copy Link
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => toggleLinkVisibility(invitation.token)}
                            >
                              {isLinkExpanded ? (
                                <ChevronUp className="size-4 mr-1" />
                              ) : (
                                <Link className="size-4 mr-1" />
                              )}
                              {isLinkExpanded ? 'Hide' : 'Show'} Link
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleResendInvitation(invitation.id)}
                              disabled={resendInvitationMutation.isPending}
                            >
                              <RefreshCw className="size-4 mr-1" />
                              Resend
                            </Button>
                          </>
                        )}
                        <ConfirmDialog
                          trigger={
                            <Button variant="ghost" size="icon">
                              <Trash2 className="size-4" />
                            </Button>
                          }
                          title="Delete invitation"
                          description={`Are you sure you want to delete the invitation for "${invitation.username}"?`}
                          confirmLabel="Delete"
                          variant="destructive"
                          onConfirm={() => handleDeleteInvitation(invitation.id)}
                        />
                      </div>
                    </CardHeader>
                    {isLinkExpanded && !invitation.usedAt && (
                      <div className="px-6 pb-4">
                        <div className="flex items-center gap-2">
                          <Input
                            readOnly
                            value={getInvitationLink(invitation.token)}
                            className="font-mono text-xs"
                            onFocus={(e) => e.target.select()}
                          />
                        </div>
                        <p className="text-xs text-muted-foreground mt-2">
                          Select the link above and copy it manually
                        </p>
                      </div>
                    )}
                  </Card>
                )
              })}
            </div>
          )}
        </TabsContent>
      </Tabs>

      <Dialog open={showInviteDialog} onOpenChange={setShowInviteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Invite User</DialogTitle>
            <DialogDescription>
              Create an invitation for a new user to join the request portal. The name you enter will become their username.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                type="text"
                placeholder="John"
                value={inviteName}
                onChange={(e) => setInviteName(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label>Quality Profile</Label>
              <Select
                value={inviteQualityProfileId?.toString() || ''}
                onValueChange={(value) => setInviteQualityProfileId(value ? parseInt(value, 10) : null)}
              >
                <SelectTrigger>
                  {inviteQualityProfileId
                    ? qualityProfiles?.find((p) => p.id === inviteQualityProfileId)?.name || 'Select profile'
                    : 'Default (use global)'
                  }
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">Default (use global)</SelectItem>
                  {qualityProfiles?.map((profile) => (
                    <SelectItem key={profile.id} value={profile.id.toString()}>
                      {profile.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="inviteAutoApprove"
                checked={inviteAutoApprove}
                onCheckedChange={(checked) => setInviteAutoApprove(checked === true)}
              />
              <Label htmlFor="inviteAutoApprove">Auto-approve requests</Label>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowInviteDialog(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreateInvitation} disabled={createInvitationMutation.isPending}>
              {createInvitationMutation.isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
              Create Invitation
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {editingUser && (
        <UserEditDialog
          user={editingUser}
          open={showUserDialog}
          onOpenChange={setShowUserDialog}
          qualityProfiles={qualityProfiles || []}
        />
      )}
    </div>
  )
}

interface UserEditDialogProps {
  user: PortalUserWithQuota
  open: boolean
  onOpenChange: (open: boolean) => void
  qualityProfiles: { id: number; name: string }[]
}

function UserEditDialog({ user, open, onOpenChange, qualityProfiles }: UserEditDialogProps) {
  const updateMutation = useUpdateAdminUser()

  const [qualityProfileId, setQualityProfileId] = useState<number | null>(user.qualityProfileId)
  const [autoApprove, setAutoApprove] = useState(user.autoApprove)
  const [useQuotaOverride, setUseQuotaOverride] = useState(
    user.quota?.moviesLimit !== null || user.quota?.seasonsLimit !== null || user.quota?.episodesLimit !== null
  )
  const [moviesLimit, setMoviesLimit] = useState(user.quota?.moviesLimit?.toString() || '')
  const [seasonsLimit, setSeasonsLimit] = useState(user.quota?.seasonsLimit?.toString() || '')
  const [episodesLimit, setEpisodesLimit] = useState(user.quota?.episodesLimit?.toString() || '')

  const handleSave = async () => {
    try {
      const input: AdminUpdateUserInput = {
        qualityProfileId,
        autoApprove,
      }

      if (useQuotaOverride) {
        input.quotaOverride = {
          moviesLimit: moviesLimit ? parseInt(moviesLimit, 10) : null,
          seasonsLimit: seasonsLimit ? parseInt(seasonsLimit, 10) : null,
          episodesLimit: episodesLimit ? parseInt(episodesLimit, 10) : null,
        }
      } else {
        input.quotaOverride = {
          moviesLimit: null,
          seasonsLimit: null,
          episodesLimit: null,
        }
      }

      await updateMutation.mutateAsync({ id: user.id, data: input })
      toast.success('User updated')
      onOpenChange(false)
    } catch {
      toast.error('Failed to update user')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Edit User</DialogTitle>
          <DialogDescription>
            Configure settings for {user.displayName || user.username}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Quality Profile</Label>
            <Select
              value={qualityProfileId?.toString() || ''}
              onValueChange={(value) => setQualityProfileId(value ? parseInt(value, 10) : null)}
            >
              <SelectTrigger>
                {qualityProfileId
                  ? qualityProfiles.find((p) => p.id === qualityProfileId)?.name || 'Select profile'
                  : 'Default (use global)'
                }
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">Default (use global)</SelectItem>
                {qualityProfiles.map((profile) => (
                  <SelectItem key={profile.id} value={profile.id.toString()}>
                    {profile.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center space-x-2">
            <Checkbox
              id="autoApprove"
              checked={autoApprove}
              onCheckedChange={(checked) => setAutoApprove(checked === true)}
            />
            <Label htmlFor="autoApprove">Auto-approve requests</Label>
          </div>

          <div className="space-y-2">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="quotaOverride"
                checked={useQuotaOverride}
                onCheckedChange={(checked) => setUseQuotaOverride(checked === true)}
              />
              <Label htmlFor="quotaOverride">Override quota limits</Label>
            </div>

            {useQuotaOverride && (
              <div className="ml-6 space-y-2 pt-2">
                <div className="grid grid-cols-3 gap-2">
                  <div className="space-y-1">
                    <Label className="text-xs">Movies</Label>
                    <Input
                      type="number"
                      placeholder="Default"
                      value={moviesLimit}
                      onChange={(e) => setMoviesLimit(e.target.value)}
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Seasons</Label>
                    <Input
                      type="number"
                      placeholder="Default"
                      value={seasonsLimit}
                      onChange={(e) => setSeasonsLimit(e.target.value)}
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Episodes</Label>
                    <Input
                      type="number"
                      placeholder="Default"
                      value={episodesLimit}
                      onChange={(e) => setEpisodesLimit(e.target.value)}
                    />
                  </div>
                </div>
                <p className="text-xs text-muted-foreground">
                  Leave empty to use the global default. Set to 0 for no limit.
                </p>
              </div>
            )}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={updateMutation.isPending}>
            {updateMutation.isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
            Save Changes
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

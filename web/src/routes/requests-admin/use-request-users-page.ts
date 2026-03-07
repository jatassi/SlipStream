import { useState } from 'react'

import { toast } from 'sonner'

import {
  getInvitationLink,
  useAdminInvitations,
  useAdminResendInvitation,
  useAdminUsers,
  useCreateInvitation,
  useDeleteAdminUser,
  useDeleteInvitation,
  useDisableUser,
  useEnableUser,
  usePortalEnabled,
  useQualityProfiles,
} from '@/hooks'
import { useUIStore } from '@/stores'
import type { PortalUserWithQuota } from '@/types'

function useUserActions() {
  const [showUserDialog, setShowUserDialog] = useState(false)
  const [editingUser, setEditingUser] = useState<PortalUserWithQuota | null>(null)

  const enableMutation = useEnableUser()
  const disableMutation = useDisableUser()
  const deleteMutation = useDeleteAdminUser()

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

  return {
    showUserDialog,
    setShowUserDialog,
    editingUser,
    togglePending: enableMutation.isPending || disableMutation.isPending,
    handleToggleEnabled,
    handleOpenEdit,
    handleDeleteUser,
  }
}

function useClipboardLink() {
  const [copiedToken, setCopiedToken] = useState<string | null>(null)
  const [expandedLinkToken, setExpandedLinkToken] = useState<string | null>(null)

  const handleCopyLink = async (token: string) => {
    const link = getInvitationLink(token)
    try {
      await navigator.clipboard.writeText(link)
      setCopiedToken(token)
      toast.success('Invitation link copied to clipboard')
      setTimeout(() => setCopiedToken(null), 3000)
      return
    } catch {
      setExpandedLinkToken(token)
    }
    toast.info('Select and copy the link below')
  }

  const toggleLinkVisibility = (token: string) => {
    setExpandedLinkToken(expandedLinkToken === token ? null : token)
  }

  return { copiedToken, expandedLinkToken, handleCopyLink, toggleLinkVisibility }
}

function useInviteDialogState() {
  const [showInviteDialog, setShowInviteDialog] = useState(false)
  const [inviteName, setInviteName] = useState('')
  const [inviteModuleSettings, setInviteModuleSettings] = useState<Record<string, number | null>>({})
  const [inviteAutoApprove, setInviteAutoApprove] = useState(false)

  const reset = () => {
    setInviteName('')
    setInviteModuleSettings({})
    setInviteAutoApprove(false)
  }

  const handleOpenInvite = () => {
    reset()
    setShowInviteDialog(true)
  }

  const setInviteModuleProfile = (moduleType: string, profileId: number | null) => {
    setInviteModuleSettings((prev) => ({ ...prev, [moduleType]: profileId }))
  }

  return {
    showInviteDialog,
    setShowInviteDialog,
    inviteName,
    setInviteName,
    inviteModuleSettings,
    setInviteModuleProfile,
    inviteAutoApprove,
    setInviteAutoApprove,
    reset,
    handleOpenInvite,
  }
}

function useInvitationActions(
  dialog: ReturnType<typeof useInviteDialogState>,
  handleCopyLink: (token: string) => Promise<void>,
) {
  const createMutation = useCreateInvitation()
  const deleteMutation = useDeleteInvitation()
  const resendMutation = useAdminResendInvitation()

  const handleCreateInvitation = async () => {
    if (!dialog.inviteName.trim()) {
      toast.error('Name is required')
      return
    }
    try {
      const params = {
        username: dialog.inviteName,
        moduleSettings: dialog.inviteModuleSettings,
        autoApprove: dialog.inviteAutoApprove,
      }
      const invitation = await createMutation.mutateAsync(params)
      toast.success('Invitation created')
      dialog.setShowInviteDialog(false)
      dialog.reset()
      void handleCopyLink(invitation.token)
    } catch (error) {
      const desc = error instanceof Error ? error.message : 'Unknown error'
      toast.error('Failed to create invitation', { description: desc })
    }
  }

  const handleDeleteInvitation = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
      toast.success('Invitation deleted')
    } catch {
      toast.error('Failed to delete invitation')
    }
  }

  const handleResendInvitation = async (id: number) => {
    try {
      const invitation = await resendMutation.mutateAsync(id)
      toast.success('Invitation resent')
      void handleCopyLink(invitation.token)
    } catch {
      toast.error('Failed to resend invitation')
    }
  }

  return { createMutation, resendMutation, handleCreateInvitation, handleDeleteInvitation, handleResendInvitation }
}

type TabQueryStateParams = {
  activeTab: string
  usersQuery: ReturnType<typeof useAdminUsers>
  invitationsQuery: ReturnType<typeof useAdminInvitations>
  globalLoading: boolean
}

function useTabQueryState({ activeTab, usersQuery, invitationsQuery, globalLoading }: TabQueryStateParams) {
  const activeQuery = activeTab === 'users' ? usersQuery : invitationsQuery
  return {
    isLoading: activeQuery.isLoading || globalLoading,
    isError: activeQuery.isError,
    refetch: activeQuery.refetch,
  }
}

export function useRequestUsersPage() {
  const [activeTab, setActiveTab] = useState<string>('users')

  const globalLoading = useUIStore((s) => s.globalLoading)
  const usersQuery = useAdminUsers()
  const invitationsQuery = useAdminInvitations()
  const { data: qualityProfiles } = useQualityProfiles()
  const portalEnabled = usePortalEnabled()

  const userActions = useUserActions()
  const clipboardLink = useClipboardLink()
  const dialogState = useInviteDialogState()
  const invitationActions = useInvitationActions(dialogState, clipboardLink.handleCopyLink)

  const tabState = useTabQueryState({ activeTab, usersQuery, invitationsQuery, globalLoading })

  const users = usersQuery.data
  const invitations = invitationsQuery.data

  return {
    activeTab,
    setActiveTab,
    users,
    invitations,
    qualityProfiles,
    portalEnabled,
    userCount: users?.length ?? 0,
    pendingInvitationCount: invitations?.filter((i) => !i.usedAt).length ?? 0,
    ...tabState,
    ...userActions,
    ...clipboardLink,
    ...dialogState,
    ...invitationActions,
  }
}

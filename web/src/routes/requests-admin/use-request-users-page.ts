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
  const [showUserSheet, setShowUserSheet] = useState(false)
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
    setShowUserSheet(true)
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
    showUserSheet,
    setShowUserSheet,
    editingUser,
    togglePending: enableMutation.isPending || disableMutation.isPending,
    handleToggleEnabled,
    handleOpenEdit,
    handleDeleteUser,
  }
}

async function copyInvitationLink(token: string) {
  try {
    await navigator.clipboard.writeText(getInvitationLink(token))
    toast.success('Invitation link copied to clipboard')
  } catch {
    toast.error('Could not copy the invitation link')
  }
}

function useInviteFormState() {
  const [showInviteSheet, setShowInviteSheet] = useState(false)
  const [inviteName, setInviteName] = useState('')
  const [inviteModuleSettings, setInviteModuleSettings] = useState<Record<string, number | null>>({})
  const [inviteAutoApprove, setInviteAutoApprove] = useState(false)

  const reset = () => {
    setInviteName('')
    setInviteModuleSettings({})
    setInviteAutoApprove(false)
  }

  return {
    showInviteSheet,
    setShowInviteSheet,
    inviteName,
    setInviteName,
    inviteModuleSettings,
    setInviteModuleProfile: (moduleType: string, profileId: number | null) => {
      setInviteModuleSettings((prev) => ({ ...prev, [moduleType]: profileId }))
    },
    inviteAutoApprove,
    setInviteAutoApprove,
    reset,
    handleOpenInvite: () => {
      reset()
      setShowInviteSheet(true)
    },
  }
}

function useCreateInvitationAction(form: ReturnType<typeof useInviteFormState>) {
  const createMutation = useCreateInvitation()

  const handleCreateInvitation = async () => {
    if (!form.inviteName.trim()) {
      toast.error('Name is required')
      return
    }
    try {
      const invitation = await createMutation.mutateAsync({
        username: form.inviteName,
        moduleSettings: form.inviteModuleSettings,
        autoApprove: form.inviteAutoApprove,
      })
      toast.success('Invitation created')
      form.setShowInviteSheet(false)
      form.reset()
      void copyInvitationLink(invitation.token)
    } catch (error) {
      const desc = error instanceof Error ? error.message : 'Unknown error'
      toast.error('Failed to create invitation', { description: desc })
    }
  }

  return { createMutation, handleCreateInvitation }
}

function useInvitationActions(form: ReturnType<typeof useInviteFormState>) {
  const deleteMutation = useDeleteInvitation()
  const resendMutation = useAdminResendInvitation()
  const create = useCreateInvitationAction(form)

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
      void copyInvitationLink(invitation.token)
    } catch {
      toast.error('Failed to resend invitation')
    }
  }

  return {
    ...create,
    resendPending: resendMutation.isPending,
    handleDeleteInvitation,
    handleResendInvitation,
    handleCopyLink: (token: string) => {
      void copyInvitationLink(token)
    },
  }
}

export function useRequestUsersPage() {
  const globalLoading = useUIStore((s) => s.globalLoading)
  const usersQuery = useAdminUsers()
  const invitationsQuery = useAdminInvitations()
  const { data: qualityProfiles } = useQualityProfiles()
  const portalEnabled = usePortalEnabled()

  const userActions = useUserActions()
  const inviteForm = useInviteFormState()
  const invitationActions = useInvitationActions(inviteForm)

  const users = usersQuery.data
  const invitations = invitationsQuery.data

  return {
    users,
    invitations,
    qualityProfiles,
    portalEnabled,
    usersState: {
      isLoading: usersQuery.isLoading || globalLoading,
      isError: usersQuery.isError,
      isEmpty: !users?.length,
      refetch: usersQuery.refetch,
    },
    invitationsState: {
      isLoading: invitationsQuery.isLoading || globalLoading,
      isError: invitationsQuery.isError,
      isEmpty: !invitations?.length,
      refetch: invitationsQuery.refetch,
    },
    ...userActions,
    ...inviteForm,
    ...invitationActions,
  }
}

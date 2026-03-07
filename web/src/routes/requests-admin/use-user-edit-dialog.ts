import { useState } from 'react'

import { toast } from 'sonner'

import { useUpdateAdminUser } from '@/hooks'
import type { AdminUpdateUserInput, PortalUserWithQuota } from '@/types'

function getInitialModuleSettings(user: PortalUserWithQuota): Record<string, number | null> {
  const result: Record<string, number | null> = {}
  for (const ms of user.moduleSettings) {
    result[ms.moduleType] = ms.qualityProfileId
  }
  return result
}

export function useUserEditDialog(
  user: PortalUserWithQuota,
  onOpenChange: (open: boolean) => void,
) {
  const updateMutation = useUpdateAdminUser()
  const [username, setUsername] = useState(user.username)
  const [moduleProfileSettings, setModuleProfileSettings] = useState<Record<string, number | null>>(
    getInitialModuleSettings(user),
  )
  const [autoApprove, setAutoApprove] = useState(user.autoApprove)

  const setModuleProfile = (moduleType: string, profileId: number | null) => {
    setModuleProfileSettings((prev) => ({ ...prev, [moduleType]: profileId }))
  }

  const handleSave = async () => {
    const input: AdminUpdateUserInput = {
      username: username === user.username ? undefined : username,
      moduleSettings: moduleProfileSettings,
      autoApprove,
    }
    try {
      await updateMutation.mutateAsync({ id: user.id, data: input })
      toast.success('User updated')
      onOpenChange(false)
    } catch {
      toast.error('Failed to update user')
    }
  }

  return {
    username, setUsername,
    moduleProfileSettings,
    setModuleProfile,
    autoApprove, setAutoApprove,
    isPending: updateMutation.isPending,
    handleSave,
  }
}

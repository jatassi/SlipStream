import { useState } from 'react'

import { toast } from 'sonner'

import { useUpdateAdminUser } from '@/hooks'
import type { AdminUpdateUserInput, PortalUserWithQuota, UserQuota } from '@/types'

function limitToString(value: number | null | undefined): string {
  return value !== null && value !== undefined ? value.toString() : ''
}

function useQuotaState(quota: UserQuota | null) {
  const hasOverride =
    quota !== null &&
    (quota.moviesLimit !== null || quota.seasonsLimit !== null || quota.episodesLimit !== null)

  const [useQuotaOverride, setUseQuotaOverride] = useState(hasOverride)
  const [moviesLimit, setMoviesLimit] = useState(limitToString(quota?.moviesLimit))
  const [seasonsLimit, setSeasonsLimit] = useState(limitToString(quota?.seasonsLimit))
  const [episodesLimit, setEpisodesLimit] = useState(limitToString(quota?.episodesLimit))

  const buildOverride = () =>
    useQuotaOverride
      ? {
          moviesLimit: moviesLimit ? Number.parseInt(moviesLimit, 10) : null,
          seasonsLimit: seasonsLimit ? Number.parseInt(seasonsLimit, 10) : null,
          episodesLimit: episodesLimit ? Number.parseInt(episodesLimit, 10) : null,
        }
      : { moviesLimit: null, seasonsLimit: null, episodesLimit: null }

  return {
    useQuotaOverride, setUseQuotaOverride,
    moviesLimit, setMoviesLimit,
    seasonsLimit, setSeasonsLimit,
    episodesLimit, setEpisodesLimit,
    buildOverride,
  }
}

export function useUserEditDialog(
  user: PortalUserWithQuota,
  onOpenChange: (open: boolean) => void,
) {
  const updateMutation = useUpdateAdminUser()
  const [username, setUsername] = useState(user.username)
  const [qualityProfileId, setQualityProfileId] = useState<number | null>(user.qualityProfileId)
  const [autoApprove, setAutoApprove] = useState(user.autoApprove)
  const quotaState = useQuotaState(user.quota)

  const handleSave = async () => {
    const input: AdminUpdateUserInput = {
      username: username === user.username ? undefined : username,
      qualityProfileId,
      autoApprove,
      quotaOverride: quotaState.buildOverride(),
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
    qualityProfileId, setQualityProfileId,
    autoApprove, setAutoApprove,
    ...quotaState,
    isPending: updateMutation.isPending,
    handleSave,
  }
}

import { useState } from 'react'

import { toast } from 'sonner'

import { useQualityProfiles } from '@/hooks'

type UseMediaEditDialogParams<T extends { id: number; monitored: boolean; qualityProfileId: number }> = {
  item: T
  updateMutation: {
    mutateAsync: (args: { id: number; data: { monitored: boolean; qualityProfileId: number } }) => Promise<unknown>
    isPending: boolean
  }
  mediaLabel: string
  onOpenChange: (open: boolean) => void
}

export function useMediaEditDialog<T extends { id: number; monitored: boolean; qualityProfileId: number }>({
  item,
  updateMutation,
  mediaLabel,
  onOpenChange,
}: UseMediaEditDialogParams<T>) {
  const [monitored, setMonitored] = useState(item.monitored)
  const [qualityProfileId, setQualityProfileId] = useState(item.qualityProfileId)
  const [prevItem, setPrevItem] = useState(item)

  if (item.id !== prevItem.id) {
    setPrevItem(item)
    setMonitored(item.monitored)
    setQualityProfileId(item.qualityProfileId)
  }

  const { data: profiles } = useQualityProfiles()
  const hasChanges = monitored !== item.monitored || qualityProfileId !== item.qualityProfileId

  const handleSubmit = async () => {
    if (!hasChanges) {
      onOpenChange(false)
      return
    }
    try {
      await updateMutation.mutateAsync({ id: item.id, data: { monitored, qualityProfileId } })
      toast.success(`${mediaLabel} updated`)
      onOpenChange(false)
    } catch {
      toast.error(`Failed to update ${mediaLabel.toLowerCase()}`)
    }
  }

  const handleProfileChange = (value: string) => {
    if (value) {
      setQualityProfileId(Number.parseInt(value, 10))
    }
  }

  const handleCancel = () => {
    onOpenChange(false)
  }

  return {
    monitored,
    setMonitored,
    qualityProfileId,
    profiles,
    handleProfileChange,
    handleSubmit,
    handleCancel,
    isPending: updateMutation.isPending,
  }
}

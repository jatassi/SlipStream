import { useCallback, useState } from 'react'

import { toast } from 'sonner'

import {
  useCreateNotification,
  useTestNewNotification,
  useUpdateNotification,
} from '@/hooks'
import type { CreateNotificationInput, NotifierSchema } from '@/types'

type FormActionsOptions = {
  onOpenChange: (open: boolean) => void
  onCreate?: (data: CreateNotificationInput) => Promise<void>
  onUpdate?: (id: number, data: CreateNotificationInput) => Promise<void>
  onTest?: (data: CreateNotificationInput) => Promise<{ success: boolean; message?: string }>
}

export function useFormActions({ onOpenChange, onCreate, onUpdate, onTest }: FormActionsOptions) {
  const [isTesting, setIsTesting] = useState(false)
  const [isPending, setIsPending] = useState(false)
  const createMutation = useCreateNotification()
  const updateMutation = useUpdateNotification()
  const testNewMutation = useTestNewNotification()

  const handleTest = useCallback(
    async (formData: CreateNotificationInput) => {
      setIsTesting(true)
      try {
        const result = onTest ? await onTest(formData) : await testNewMutation.mutateAsync(formData)
        toast[result.success ? 'success' : 'error'](
          result.message ?? (result.success ? 'Notification test successful' : 'Notification test failed'),
        )
      } catch {
        toast.error('Failed to test notification')
      } finally {
        setIsTesting(false)
      }
    },
    [onTest, testNewMutation],
  )

  const handleSubmit = useCallback(
    async (formData: CreateNotificationInput, schema: NotifierSchema | undefined, notification: { id: number } | null | undefined) => {
      const error = validateForm(formData, schema)
      if (error) { toast.error(error); return }
      setIsPending(true)
      try {
        await saveNotification(formData, notification, { onCreate, onUpdate, createMutation, updateMutation })
        onOpenChange(false)
      } catch {
        toast.error(notification ? 'Failed to update notification' : 'Failed to create notification')
      } finally {
        setIsPending(false)
      }
    },
    [onUpdate, updateMutation, onCreate, createMutation, onOpenChange],
  )

  return { isTesting, isPending, handleTest, handleSubmit }
}

async function saveNotification(
  formData: CreateNotificationInput,
  notification: { id: number } | null | undefined,
  deps: {
    onCreate?: (data: CreateNotificationInput) => Promise<void>
    onUpdate?: (id: number, data: CreateNotificationInput) => Promise<void>
    createMutation: ReturnType<typeof useCreateNotification>
    updateMutation: ReturnType<typeof useUpdateNotification>
  },
) {
  if (notification) {
    await (deps.onUpdate
      ? deps.onUpdate(notification.id, formData)
      : deps.updateMutation.mutateAsync({ id: notification.id, data: formData }))
    toast.success('Notification updated')
  } else {
    await (deps.onCreate ? deps.onCreate(formData) : deps.createMutation.mutateAsync(formData))
    toast.success('Notification created')
  }
}

function validateForm(
  formData: CreateNotificationInput,
  currentSchema?: { fields: { name: string; label: string; type: string; required?: boolean }[] },
): string | null {
  if (!formData.name.trim()) { return 'Name is required' }
  for (const field of currentSchema?.fields.filter((f) => f.required) ?? []) {
    if (field.type === 'action') { continue }
    const value = formData.settings[field.name]
    if (!value || (typeof value === 'string' && !value.trim())) { return `${field.label} is required` }
  }
  return null
}

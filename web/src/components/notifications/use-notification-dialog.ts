import { useCallback, useMemo, useState } from 'react'

import { useNotificationSchemas } from '@/hooks'
import type { CreateNotificationInput, NotifierType } from '@/types'

import type { NotificationDialogProps } from './notification-dialog-types'
import { adminEventTriggers, defaultFormData } from './notification-dialog-types'
import { useFormActions } from './use-form-actions'
import { useInitializeForm } from './use-initialize-form'
import { usePlexState } from './use-plex-state'

export function useNotificationDialog(props: NotificationDialogProps) {
  const { open, onOpenChange, notification, eventTriggers, schemas: customSchemas, onCreate, onUpdate, onTest } = props
  const [formData, setFormData] = useState<CreateNotificationInput>(defaultFormData)
  const [showAdvanced, setShowAdvanced] = useState(false)

  const { data: fetchedSchemas } = useNotificationSchemas()
  const schemas = customSchemas ?? fetchedSchemas
  const triggers = eventTriggers ?? adminEventTriggers
  const isEditing = !!notification
  const isPlex = formData.type === 'plex'
  const hasPlexToken = !!(formData.settings.authToken as string)
  const currentSchema = useMemo(() => schemas?.find((s) => s.type === formData.type), [schemas, formData.type])
  const hasAdvancedFields = useMemo(() => currentSchema?.fields.some((f) => f.advanced) ?? false, [currentSchema])

  const handleSettingChange = useCallback((name: string, value: unknown) => {
    setFormData((prev) => ({ ...prev, settings: { ...prev.settings, [name]: value } }))
  }, [])

  const plex = usePlexState({ isPlex, serverId: formData.settings.serverId, authToken: formData.settings.authToken, onSettingChange: handleSettingChange })
  const actions = useFormActions({ onOpenChange, onCreate, onUpdate, onTest })

  useInitializeForm({ open, notification, eventTriggers, plex, setFormData, setShowAdvanced })

  const handleTypeChange = useCallback((type: NotifierType) => {
    const schema = schemas?.find((s) => s.type === type)
    const newSettings: Record<string, unknown> = {}
    schema?.fields.forEach((field) => { if (field.default !== undefined) { newSettings[field.name] = field.default } })
    setFormData((prev) => ({ ...prev, type, settings: newSettings }))
    plex.resetState()
    plex.cleanupPolling()
  }, [schemas, plex])

  const handleFormDataChange = useCallback((key: string, value: unknown) => {
    setFormData((prev) => ({ ...prev, [key]: value }))
  }, [])

  return {
    formData, isTesting: actions.isTesting, showAdvanced, isPending: actions.isPending,
    isPlexConnecting: plex.isPlexConnecting, plexServers: plex.plexServers, plexSections: plex.plexSections,
    isLoadingServers: plex.isLoadingServers, isLoadingSections: plex.isLoadingSections,
    schemas, triggers, isEditing, currentSchema, hasAdvancedFields, isPlex, hasPlexToken,
    handleSettingChange, handlePlexOAuth: plex.startOAuth, handleTypeChange,
    handleTest: useCallback(() => actions.handleTest(formData), [actions, formData]),
    handleSubmit: useCallback(() => actions.handleSubmit(formData, currentSchema, notification), [actions, formData, currentSchema, notification]),
    handleFormDataChange, handlePlexDisconnect: plex.disconnect,
    toggleAdvanced: useCallback(() => setShowAdvanced((prev) => !prev), []),
  }
}

export type NotificationDialogState = ReturnType<typeof useNotificationDialog>

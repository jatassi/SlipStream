import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { Loader2, TestTube, ExternalLink, ChevronDown, ChevronUp, Check, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import {
  useCreateNotification,
  useUpdateNotification,
  useTestNewNotification,
  useNotificationSchemas,
} from '@/hooks'
import { apiFetch } from '@/api/client'
import type {
  Notification,
  NotifierType,
  CreateNotificationInput,
  SettingsField,
  NotifierSchema,
} from '@/types'

interface EventTrigger {
  key: string
  label: string
  description?: string
}

interface NotificationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  notification?: Notification | null
  /** Custom event triggers to display. If not provided, uses default admin triggers */
  eventTriggers?: EventTrigger[]
  /** Custom schemas. If not provided, fetches from API */
  schemas?: NotifierSchema[]
  /** Custom create handler. If not provided, uses admin API */
  onCreate?: (data: CreateNotificationInput) => Promise<void>
  /** Custom update handler. If not provided, uses admin API */
  onUpdate?: (id: number, data: CreateNotificationInput) => Promise<void>
  /** Custom test handler. If not provided, uses admin API */
  onTest?: (data: CreateNotificationInput) => Promise<{ success: boolean; message?: string }>
}

interface PlexServer {
  id: string
  name: string
  owned: boolean
  address?: string
}

interface PlexSection {
  key: number
  title: string
  type: string
}

const defaultFormData: CreateNotificationInput = {
  name: '',
  type: 'discord',
  enabled: true,
  settings: {},
  onGrab: true,
  onImport: true,
  onUpgrade: true,
  onMovieAdded: false,
  onMovieDeleted: false,
  onSeriesAdded: false,
  onSeriesDeleted: false,
  onHealthIssue: true,
  onHealthRestored: true,
  onAppUpdate: false,
  includeHealthWarnings: true,
  tags: [],
}

const adminEventTriggers: EventTrigger[] = [
  { key: 'onGrab', label: 'On Grab', description: 'When a release is grabbed' },
  { key: 'onImport', label: 'On Import', description: 'When a file is imported to the library' },
  { key: 'onUpgrade', label: 'On Upgrade', description: 'When a quality upgrade is imported' },
  { key: 'onMovieAdded', label: 'On Movie Added', description: 'When a movie is added' },
  { key: 'onMovieDeleted', label: 'On Movie Deleted', description: 'When a movie is removed' },
  { key: 'onSeriesAdded', label: 'On Series Added', description: 'When a series is added' },
  { key: 'onSeriesDeleted', label: 'On Series Deleted', description: 'When a series is removed' },
  { key: 'onHealthIssue', label: 'On Health Issue', description: 'When a health check fails' },
  { key: 'onHealthRestored', label: 'On Health Restored', description: 'When a health issue is resolved' },
  { key: 'onAppUpdate', label: 'On App Update', description: 'When the application is updated' },
]

export function NotificationDialog({
  open,
  onOpenChange,
  notification,
  eventTriggers,
  schemas: customSchemas,
  onCreate,
  onUpdate,
  onTest,
}: NotificationDialogProps) {
  const [formData, setFormData] = useState<CreateNotificationInput>(defaultFormData)
  const [isTesting, setIsTesting] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [isPending, setIsPending] = useState(false)

  // Plex OAuth state
  const [, setPlexPinId] = useState<number | null>(null)
  const [isPlexConnecting, setIsPlexConnecting] = useState(false)
  const [plexServers, setPlexServers] = useState<PlexServer[]>([])
  const [plexSections, setPlexSections] = useState<PlexSection[]>([])
  const [isLoadingServers, setIsLoadingServers] = useState(false)
  const [isLoadingSections, setIsLoadingSections] = useState(false)
  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const { data: fetchedSchemas } = useNotificationSchemas()
  const createMutation = useCreateNotification()
  const updateMutation = useUpdateNotification()
  const testNewMutation = useTestNewNotification()

  const schemas = customSchemas ?? fetchedSchemas
  const triggers = eventTriggers ?? adminEventTriggers
  const isEditing = !!notification

  const currentSchema = useMemo(() => {
    return schemas?.find((s) => s.type === formData.type)
  }, [schemas, formData.type])

  const hasAdvancedFields = useMemo(() => {
    return currentSchema?.fields.some((f) => f.advanced) ?? false
  }, [currentSchema])

  const isPlex = formData.type === 'plex'
  const hasPlexToken = !!(formData.settings.authToken as string)

  const cleanupPlexPolling = useCallback(() => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current)
      pollIntervalRef.current = null
    }
    setPlexPinId(null)
    setIsPlexConnecting(false)
  }, [])

  useEffect(() => {
    if (open) {
      if (notification) {
        setFormData({
          name: notification.name,
          type: notification.type,
          enabled: notification.enabled,
          settings: notification.settings || {},
          onGrab: notification.onGrab,
          onImport: notification.onImport,
          onUpgrade: notification.onUpgrade,
          onMovieAdded: notification.onMovieAdded,
          onMovieDeleted: notification.onMovieDeleted,
          onSeriesAdded: notification.onSeriesAdded,
          onSeriesDeleted: notification.onSeriesDeleted,
          onHealthIssue: notification.onHealthIssue,
          onHealthRestored: notification.onHealthRestored,
          onAppUpdate: notification.onAppUpdate,
          includeHealthWarnings: notification.includeHealthWarnings,
          tags: notification.tags || [],
        })
        // If editing Plex with existing token, load servers
        if (notification.type === 'plex' && notification.settings.authToken) {
          fetchPlexServers(notification.settings.authToken as string)
        }
      } else {
        const resetData = { ...defaultFormData }
        if (eventTriggers) {
          adminEventTriggers.forEach(t => {
            (resetData as unknown as Record<string, unknown>)[t.key] = false
          })
          eventTriggers.forEach(t => {
            (resetData as unknown as Record<string, unknown>)[t.key] = true
          })
        }
        setFormData(resetData)
      }
      setShowAdvanced(false)
      setPlexServers([])
      setPlexSections([])
    } else {
      cleanupPlexPolling()
    }
  }, [open, notification, eventTriggers, cleanupPlexPolling])

  // Load sections when server changes
  useEffect(() => {
    const serverId = formData.settings.serverId as string
    const token = formData.settings.authToken as string
    if (isPlex && serverId && token) {
      fetchPlexSections(serverId, token)
    }
  }, [isPlex, formData.settings.serverId, formData.settings.authToken])

  const fetchPlexServers = async (token: string) => {
    setIsLoadingServers(true)
    try {
      const servers = await apiFetch<PlexServer[]>('/notifications/plex/servers', {
        headers: { 'X-Plex-Token': token },
      })
      setPlexServers(servers)
    } catch (err) {
      console.error('Failed to fetch Plex servers:', err)
    } finally {
      setIsLoadingServers(false)
    }
  }

  const fetchPlexSections = async (serverId: string, token: string) => {
    setIsLoadingSections(true)
    try {
      const sections = await apiFetch<PlexSection[]>(`/notifications/plex/servers/${serverId}/sections`, {
        headers: { 'X-Plex-Token': token },
      })
      setPlexSections(sections)
    } catch (err) {
      console.error('Failed to fetch Plex sections:', err)
    } finally {
      setIsLoadingSections(false)
    }
  }

  const handlePlexOAuth = async () => {
    setIsPlexConnecting(true)
    try {
      const { pinId, authUrl } = await apiFetch<{ pinId: number; authUrl: string }>('/notifications/plex/auth/start', {
        method: 'POST',
      })
      setPlexPinId(pinId)

      // Open auth URL in new window
      window.open(authUrl, '_blank', 'width=800,height=600')

      // Poll for completion
      pollIntervalRef.current = setInterval(async () => {
        try {
          const status = await apiFetch<{ complete: boolean; authToken?: string }>(`/notifications/plex/auth/status/${pinId}`)
          if (status.complete && status.authToken) {
            cleanupPlexPolling()
            handleSettingChange('authToken', status.authToken)
            toast.success('Connected to Plex!')
            fetchPlexServers(status.authToken)
          }
        } catch (err) {
          // Check if it's a 410 Gone (expired)
          if (err && typeof err === 'object' && 'status' in err && err.status === 410) {
            cleanupPlexPolling()
            toast.error('Plex authentication expired. Please try again.')
          }
          // Ignore other polling errors
        }
      }, 2000)

      // Timeout after 5 minutes
      setTimeout(() => {
        if (pollIntervalRef.current) {
          cleanupPlexPolling()
          toast.error('Plex authentication timed out. Please try again.')
        }
      }, 5 * 60 * 1000)
    } catch {
      setIsPlexConnecting(false)
      toast.error('Failed to start Plex authentication')
    }
  }

  const handleTypeChange = (type: NotifierType) => {
    const schema = schemas?.find((s) => s.type === type)
    const newSettings: Record<string, unknown> = {}

    schema?.fields.forEach((field) => {
      if (field.default !== undefined) {
        newSettings[field.name] = field.default
      }
    })

    setFormData((prev) => ({
      ...prev,
      type,
      settings: newSettings,
    }))

    // Reset Plex state when switching types
    setPlexServers([])
    setPlexSections([])
    cleanupPlexPolling()
  }

  const handleSettingChange = (name: string, value: unknown) => {
    setFormData((prev) => ({
      ...prev,
      settings: {
        ...prev.settings,
        [name]: value,
      },
    }))
  }

  const handleTest = async () => {
    setIsTesting(true)
    try {
      if (onTest) {
        const result = await onTest(formData)
        if (result.success) {
          toast.success(result.message || 'Notification test successful')
        } else {
          toast.error(result.message || 'Notification test failed')
        }
      } else {
        const result = await testNewMutation.mutateAsync(formData)
        if (result.success) {
          toast.success(result.message || 'Notification test successful')
        } else {
          toast.error(result.message || 'Notification test failed')
        }
      }
    } catch {
      toast.error('Failed to test notification')
    } finally {
      setIsTesting(false)
    }
  }

  const handleSubmit = async () => {
    if (!formData.name.trim()) {
      toast.error('Name is required')
      return
    }

    const requiredFields = currentSchema?.fields.filter((f) => f.required) || []
    for (const field of requiredFields) {
      if (field.type === 'action') continue // Skip action fields in validation
      const value = formData.settings[field.name]
      if (!value || (typeof value === 'string' && !value.trim())) {
        toast.error(`${field.label} is required`)
        return
      }
    }

    setIsPending(true)
    try {
      if (isEditing && notification) {
        if (onUpdate) {
          await onUpdate(notification.id, formData)
        } else {
          await updateMutation.mutateAsync({ id: notification.id, data: formData })
        }
        toast.success('Notification updated')
      } else {
        if (onCreate) {
          await onCreate(formData)
        } else {
          await createMutation.mutateAsync(formData)
        }
        toast.success('Notification created')
      }
      onOpenChange(false)
    } catch {
      toast.error(isEditing ? 'Failed to update notification' : 'Failed to create notification')
    } finally {
      setIsPending(false)
    }
  }

  const renderField = (field: SettingsField) => {
    const value = formData.settings[field.name]

    switch (field.type) {
      case 'text':
      case 'url':
        return (
          <Input
            id={field.name}
            type={field.type === 'url' ? 'url' : 'text'}
            placeholder={field.placeholder}
            value={(value as string) || ''}
            onChange={(e) => handleSettingChange(field.name, e.target.value)}
          />
        )

      case 'password':
        return (
          <Input
            id={field.name}
            type="password"
            placeholder={field.placeholder}
            value={(value as string) || ''}
            onChange={(e) => handleSettingChange(field.name, e.target.value)}
          />
        )

      case 'number':
        return (
          <Input
            id={field.name}
            type="number"
            placeholder={field.placeholder}
            value={value !== undefined ? String(value) : ''}
            onChange={(e) => handleSettingChange(field.name, e.target.value ? Number(e.target.value) : undefined)}
          />
        )

      case 'bool':
        return (
          <Switch
            id={field.name}
            checked={Boolean(value)}
            onCheckedChange={(checked) => handleSettingChange(field.name, checked)}
          />
        )

      case 'select':
        // Special handling for Plex server select
        if (isPlex && field.name === 'serverId') {
          return (
            <Select
              value={(value as string) || ''}
              onValueChange={(v) => handleSettingChange(field.name, v)}
              disabled={!hasPlexToken || isLoadingServers}
            >
              <SelectTrigger>
                {isLoadingServers ? (
                  <span className="flex items-center gap-2">
                    <Loader2 className="size-4 animate-spin" />
                    Loading servers...
                  </span>
                ) : (
                  plexServers.find((s) => s.id === value)?.name || 'Select server...'
                )}
              </SelectTrigger>
              <SelectContent>
                {plexServers.map((server) => (
                  <SelectItem key={server.id} value={server.id}>
                    {server.name} {server.owned && '(owned)'}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )
        }

        return (
          <Select
            value={(value as string) || field.default as string || ''}
            onValueChange={(v) => handleSettingChange(field.name, v)}
          >
            <SelectTrigger>
              {field.options?.find((o) => o.value === (value || field.default))?.label || 'Select...'}
            </SelectTrigger>
            <SelectContent>
              {field.options?.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )

      case 'action':
        if (field.actionType === 'oauth' && isPlex) {
          return (
            <div className="flex items-center gap-2">
              {hasPlexToken ? (
                <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
                  <Check className="size-4" />
                  Connected
                </div>
              ) : isPlexConnecting ? (
                <Button variant="outline" disabled>
                  <Loader2 className="size-4 mr-2 animate-spin" />
                  Waiting for approval...
                </Button>
              ) : (
                <Button variant="outline" onClick={handlePlexOAuth}>
                  {field.actionLabel || 'Connect'}
                </Button>
              )}
              {hasPlexToken && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    handleSettingChange('authToken', '')
                    handleSettingChange('serverId', '')
                    handleSettingChange('sectionIds', [])
                    setPlexServers([])
                    setPlexSections([])
                  }}
                >
                  <X className="size-4" />
                </Button>
              )}
            </div>
          )
        }
        return (
          <Button variant="outline" onClick={() => field.actionEndpoint && fetch(field.actionEndpoint)}>
            {field.actionLabel || 'Action'}
          </Button>
        )

      default:
        return null
    }
  }

  // Render Plex sections selector (custom field not in schema)
  const renderPlexSections = () => {
    if (!isPlex || !hasPlexToken || !formData.settings.serverId) return null

    const currentSectionIds = (formData.settings.sectionIds as number[]) || []

    return (
      <div className="space-y-2">
        <Label>Library Sections</Label>
        <p className="text-xs text-muted-foreground">Select which library sections to refresh</p>
        {isLoadingSections ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" />
            Loading sections...
          </div>
        ) : plexSections.length === 0 ? (
          <p className="text-sm text-muted-foreground">No movie or TV sections found</p>
        ) : (
          <div className="space-y-2 border rounded-lg p-3">
            {plexSections.map((section) => (
              <div key={section.key} className="flex items-center justify-between">
                <Label htmlFor={`section-${section.key}`} className="font-normal cursor-pointer">
                  {section.title} ({section.type})
                </Label>
                <Switch
                  id={`section-${section.key}`}
                  checked={currentSectionIds.includes(section.key)}
                  onCheckedChange={(checked) => {
                    const newIds = checked
                      ? [...currentSectionIds, section.key]
                      : currentSectionIds.filter((id) => id !== section.key)
                    handleSettingChange('sectionIds', newIds)
                  }}
                />
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit Notification' : 'Add Notification'}</DialogTitle>
          <DialogDescription>
            {currentSchema?.description || 'Configure notification settings and triggers.'}
            {currentSchema?.infoUrl && (
              <a
                href={currentSchema.infoUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 ml-1 text-primary hover:underline"
              >
                Learn more <ExternalLink className="size-3" />
              </a>
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* Notification Type */}
          <div className="space-y-2">
            <Label htmlFor="type">Type</Label>
            <Select
              value={formData.type}
              onValueChange={(v) => handleTypeChange(v as NotifierType)}
              disabled={isEditing}
            >
              <SelectTrigger>
                {schemas?.find((s) => s.type === formData.type)?.name || formData.type}
              </SelectTrigger>
              <SelectContent>
                {schemas?.map((schema) => (
                  <SelectItem key={schema.type} value={schema.type}>
                    {schema.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Name */}
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              placeholder="My Notification"
              value={formData.name}
              onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
            />
          </div>

          {/* Provider Settings */}
          {currentSchema?.fields
            .filter((f) => !f.advanced)
            .map((field) => (
              <div key={field.name} className={field.type === 'bool' ? 'flex items-center justify-between' : 'space-y-2'}>
                <div>
                  <Label htmlFor={field.name}>
                    {field.label}
                    {!field.required && field.type !== 'bool' && field.type !== 'action' && (
                      <span className="text-muted-foreground text-xs ml-1">(optional)</span>
                    )}
                  </Label>
                  {field.helpText && field.type !== 'bool' && (
                    <p className="text-xs text-muted-foreground">{field.helpText}</p>
                  )}
                </div>
                {renderField(field)}
              </div>
            ))}

          {/* Plex Library Sections (custom rendering) */}
          {renderPlexSections()}

          {/* Advanced Settings Toggle */}
          {hasAdvancedFields && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="w-full"
              onClick={() => setShowAdvanced(!showAdvanced)}
            >
              {showAdvanced ? (
                <>
                  <ChevronUp className="size-4 mr-2" />
                  Hide Advanced Settings
                </>
              ) : (
                <>
                  <ChevronDown className="size-4 mr-2" />
                  Show Advanced Settings
                </>
              )}
            </Button>
          )}

          {/* Advanced Provider Settings */}
          {showAdvanced &&
            currentSchema?.fields
              .filter((f) => f.advanced)
              .map((field) => (
                <div key={field.name} className={field.type === 'bool' ? 'flex items-center justify-between' : 'space-y-2'}>
                  <div>
                    <Label htmlFor={field.name}>
                      {field.label}
                      {!field.required && field.type !== 'bool' && (
                        <span className="text-muted-foreground text-xs ml-1">(optional)</span>
                      )}
                    </Label>
                    {field.helpText && field.type !== 'bool' && (
                      <p className="text-xs text-muted-foreground">{field.helpText}</p>
                    )}
                  </div>
                  {renderField(field)}
                </div>
              ))}

          {/* Event Triggers */}
          {triggers.length > 0 && (
            <div className="space-y-3">
              <Label>Event Triggers</Label>
              <div className="space-y-2 border rounded-lg p-3">
                {triggers.map(({ key, label }) => (
                  <div key={key} className="flex items-center justify-between">
                    <Label htmlFor={key} className="font-normal cursor-pointer">
                      {label}
                    </Label>
                    <Switch
                      id={key}
                      checked={Boolean((formData as unknown as Record<string, unknown>)[key])}
                      onCheckedChange={(checked) =>
                        setFormData((prev) => ({ ...prev, [key]: checked }))
                      }
                    />
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Enabled Toggle */}
          <div className="flex items-center justify-between">
            <Label htmlFor="enabled">Enabled</Label>
            <Switch
              id="enabled"
              checked={formData.enabled}
              onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, enabled: checked }))}
            />
          </div>
        </div>

        <DialogFooter className="flex-col gap-2 sm:flex-row">
          <Button variant="outline" onClick={handleTest} disabled={isTesting}>
            {isTesting ? (
              <Loader2 className="size-4 mr-2 animate-spin" />
            ) : (
              <TestTube className="size-4 mr-2" />
            )}
            Test
          </Button>
          <div className="flex gap-2 sm:ml-auto">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button onClick={handleSubmit} disabled={isPending}>
              {isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
              {isEditing ? 'Save' : 'Add'}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

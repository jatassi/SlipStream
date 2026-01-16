import { useState, useEffect, useMemo } from 'react'
import { Loader2, TestTube, ExternalLink, ChevronDown, ChevronUp } from 'lucide-react'
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
import type {
  Notification,
  NotifierType,
  CreateNotificationInput,
  SettingsField,
} from '@/types'

interface NotificationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  notification?: Notification | null
}

const defaultFormData: CreateNotificationInput = {
  name: '',
  type: 'discord',
  enabled: true,
  settings: {},
  onGrab: true,
  onDownload: true,
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

const eventTriggers = [
  { key: 'onGrab', label: 'On Grab', description: 'When a release is grabbed' },
  { key: 'onDownload', label: 'On Download', description: 'When a download completes' },
  { key: 'onUpgrade', label: 'On Upgrade', description: 'When a quality upgrade is imported' },
  { key: 'onMovieAdded', label: 'On Movie Added', description: 'When a movie is added' },
  { key: 'onMovieDeleted', label: 'On Movie Deleted', description: 'When a movie is removed' },
  { key: 'onSeriesAdded', label: 'On Series Added', description: 'When a series is added' },
  { key: 'onSeriesDeleted', label: 'On Series Deleted', description: 'When a series is removed' },
  { key: 'onHealthIssue', label: 'On Health Issue', description: 'When a health check fails' },
  { key: 'onHealthRestored', label: 'On Health Restored', description: 'When a health issue is resolved' },
  { key: 'onAppUpdate', label: 'On App Update', description: 'When the application is updated' },
] as const

export function NotificationDialog({
  open,
  onOpenChange,
  notification,
}: NotificationDialogProps) {
  const [formData, setFormData] = useState<CreateNotificationInput>(defaultFormData)
  const [isTesting, setIsTesting] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(false)

  const { data: schemas } = useNotificationSchemas()
  const createMutation = useCreateNotification()
  const updateMutation = useUpdateNotification()
  const testNewMutation = useTestNewNotification()

  const isEditing = !!notification

  const currentSchema = useMemo(() => {
    return schemas?.find((s) => s.type === formData.type)
  }, [schemas, formData.type])

  const hasAdvancedFields = useMemo(() => {
    return currentSchema?.fields.some((f) => f.advanced) ?? false
  }, [currentSchema])

  useEffect(() => {
    if (open) {
      if (notification) {
        setFormData({
          name: notification.name,
          type: notification.type,
          enabled: notification.enabled,
          settings: notification.settings || {},
          onGrab: notification.onGrab,
          onDownload: notification.onDownload,
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
      } else {
        setFormData(defaultFormData)
      }
      setShowAdvanced(false)
    }
  }, [open, notification])

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
      const result = await testNewMutation.mutateAsync(formData)
      if (result.success) {
        toast.success(result.message || 'Notification test successful')
      } else {
        toast.error(result.message || 'Notification test failed')
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
      const value = formData.settings[field.name]
      if (!value || (typeof value === 'string' && !value.trim())) {
        toast.error(`${field.label} is required`)
        return
      }
    }

    try {
      if (isEditing && notification) {
        await updateMutation.mutateAsync({
          id: notification.id,
          data: formData,
        })
        toast.success('Notification updated')
      } else {
        await createMutation.mutateAsync(formData)
        toast.success('Notification created')
      }
      onOpenChange(false)
    } catch {
      toast.error(isEditing ? 'Failed to update notification' : 'Failed to create notification')
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

      default:
        return null
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

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
          <div className="space-y-3">
            <Label>Event Triggers</Label>
            <div className="space-y-2 border rounded-lg p-3">
              {eventTriggers.map(({ key, label }) => (
                <div key={key} className="flex items-center justify-between">
                  <Label htmlFor={key} className="font-normal cursor-pointer">
                    {label}
                  </Label>
                  <Switch
                    id={key}
                    checked={Boolean(formData[key as keyof CreateNotificationInput])}
                    onCheckedChange={(checked) =>
                      setFormData((prev) => ({ ...prev, [key]: checked }))
                    }
                  />
                </div>
              ))}
            </div>
          </div>

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

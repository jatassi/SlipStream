import { useState, useEffect } from 'react'
import { Loader2, ChevronDown, AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
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
  useCreateQualityProfile,
  useUpdateQualityProfile,
  useQualityProfileAttributes,
} from '@/hooks'
import type {
  QualityProfile,
  QualityItem,
  CreateQualityProfileInput,
  AttributeSettings,
  AttributeMode,
} from '@/types'
import { PREDEFINED_QUALITIES, DEFAULT_ATTRIBUTE_SETTINGS } from '@/types'

interface QualityProfileDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  profile?: QualityProfile | null
}

const MODE_OPTIONS: { value: AttributeMode; label: string }[] = [
  { value: 'required', label: 'Required' },
  { value: 'preferred', label: 'Preferred' },
  { value: 'acceptable', label: 'Acceptable' },
  { value: 'notAllowed', label: 'Not Allowed' },
]

const MODE_LABELS: Record<AttributeMode, string> = {
  required: 'Required',
  preferred: 'Preferred',
  acceptable: 'Acceptable',
  notAllowed: 'Not Allowed',
}

const defaultItems: QualityItem[] = PREDEFINED_QUALITIES.map((q) => ({
  quality: q,
  allowed: q.weight >= 10,
}))

const defaultFormData: CreateQualityProfileInput = {
  name: '',
  cutoff: 10,
  upgradesEnabled: true,
  allowAutoApprove: false,
  items: defaultItems,
  hdrSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
  videoCodecSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
  audioCodecSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
  audioChannelSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
}

export function QualityProfileDialog({
  open,
  onOpenChange,
  profile,
}: QualityProfileDialogProps) {
  const [formData, setFormData] = useState<CreateQualityProfileInput>(defaultFormData)

  const createMutation = useCreateQualityProfile()
  const updateMutation = useUpdateQualityProfile()
  const { data: attributeOptions } = useQualityProfileAttributes()

  const isEditing = !!profile

  useEffect(() => {
    if (open) {
      if (profile) {
        setFormData({
          name: profile.name,
          cutoff: profile.cutoff,
          upgradesEnabled: profile.upgradesEnabled ?? true,
          allowAutoApprove: profile.allowAutoApprove ?? false,
          items: profile.items,
          hdrSettings: profile.hdrSettings || { ...DEFAULT_ATTRIBUTE_SETTINGS },
          videoCodecSettings: profile.videoCodecSettings || { ...DEFAULT_ATTRIBUTE_SETTINGS },
          audioCodecSettings: profile.audioCodecSettings || { ...DEFAULT_ATTRIBUTE_SETTINGS },
          audioChannelSettings: profile.audioChannelSettings || { ...DEFAULT_ATTRIBUTE_SETTINGS },
        })
      } else {
        setFormData({
          ...defaultFormData,
          items: defaultItems.map((i) => ({ ...i })),
          hdrSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
          videoCodecSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
          audioCodecSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
          audioChannelSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
        })
      }
    }
  }, [open, profile])

  const handleSubmit = async () => {
    if (!formData.name.trim()) {
      toast.error('Name is required')
      return
    }

    const allowedQualities = formData.items.filter((i) => i.allowed)
    if (allowedQualities.length === 0) {
      toast.error('At least one quality must be allowed')
      return
    }

    try {
      if (isEditing && profile) {
        await updateMutation.mutateAsync({
          id: profile.id,
          data: formData,
        })
        toast.success('Profile updated')
      } else {
        await createMutation.mutateAsync(formData)
        toast.success('Profile created')
      }
      onOpenChange(false)
    } catch {
      toast.error(isEditing ? 'Failed to update profile' : 'Failed to create profile')
    }
  }

  const toggleQuality = (qualityId: number) => {
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) =>
        item.quality.id === qualityId
          ? { ...item, allowed: !item.allowed }
          : item
      ),
    }))
  }

  const HDR_FORMATS = ['DV', 'HDR10+', 'HDR10', 'HDR', 'HLG']

  const updateItemMode = (
    field: 'hdrSettings' | 'videoCodecSettings' | 'audioCodecSettings' | 'audioChannelSettings',
    value: string,
    mode: AttributeMode
  ) => {
    setFormData((prev) => {
      const currentItems = { ...prev[field].items }
      if (mode === 'acceptable') {
        delete currentItems[value]
      } else {
        currentItems[value] = mode
      }

      // Auto-set mutual exclusivity for HDR/SDR
      if (field === 'hdrSettings' && mode === 'required') {
        if (value === 'SDR') {
          // Setting SDR to required → block all HDR formats
          for (const hdrFormat of HDR_FORMATS) {
            currentItems[hdrFormat] = 'notAllowed'
          }
        } else if (HDR_FORMATS.includes(value)) {
          // Setting an HDR format to required → block SDR
          currentItems['SDR'] = 'notAllowed'
        }
      }

      return {
        ...prev,
        [field]: { items: currentItems },
      }
    })
  }

  const isPending = createMutation.isPending || updateMutation.isPending
  const allowedQualities = formData.items.filter((i) => i.allowed)
  const cutoffOptions = allowedQualities.length > 0 ? allowedQualities : formData.items

  // Calculate disabled HDR items based on mutual exclusivity
  const getDisabledHdrItems = (): string[] => {
    const disabled: string[] = []
    const hdrItems = formData.hdrSettings.items

    // If SDR is required, disable all HDR formats
    if (hdrItems['SDR'] === 'required') {
      disabled.push(...HDR_FORMATS)
    }

    // If any HDR format is required, disable SDR
    const hasRequiredHdr = HDR_FORMATS.some((f) => hdrItems[f] === 'required')
    if (hasRequiredHdr) {
      disabled.push('SDR')
    }

    return disabled
  }

  const disabledHdrItems = getDisabledHdrItems()

  // Validate attribute groups - all items same non-any mode is invalid
  const validateAttributeGroup = (settings: AttributeSettings, options: string[]): string | null => {
    if (options.length === 0) return null

    const modes = options.map((opt) => settings.items[opt] || 'acceptable')
    const nonAcceptableModes = modes.filter((m) => m !== 'acceptable')

    // If not all items have a non-acceptable mode set, it's valid
    if (nonAcceptableModes.length !== options.length) return null

    // Check if all are the same mode
    const firstMode = nonAcceptableModes[0]
    const allSame = nonAcceptableModes.every((m) => m === firstMode)

    if (allSame) {
      switch (firstMode) {
        case 'required':
          return 'All items set to Required - no release can match all requirements'
        case 'preferred':
          return 'All items set to Preferred - this is equivalent to Acceptable'
        case 'notAllowed':
          return 'All items set to Not Allowed - no release can match'
      }
    }
    return null
  }

  const hdrOptions = ['SDR', ...(attributeOptions?.hdrFormats || []).filter((f) => f !== 'SDR')]
  const attributeValidation = {
    hdr: validateAttributeGroup(formData.hdrSettings, hdrOptions),
    videoCodec: validateAttributeGroup(formData.videoCodecSettings, attributeOptions?.videoCodecs || []),
    audioCodec: validateAttributeGroup(formData.audioCodecSettings, attributeOptions?.audioCodecs || []),
    audioChannels: validateAttributeGroup(formData.audioChannelSettings, attributeOptions?.audioChannels || []),
  }

  const hasAttributeValidationError = Object.values(attributeValidation).some((v) => v !== null)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit Quality Profile' : 'Add Quality Profile'}</DialogTitle>
          <DialogDescription>
            Configure quality preferences and attribute filters for downloads.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* Name */}
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              placeholder="HD-1080p"
              value={formData.name}
              onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
            />
          </div>

          {/* Qualities - Grouped by resolution */}
          <div className="space-y-2">
            <Label>Allowed Qualities</Label>
            <div className="border rounded-lg bg-muted/30 divide-y">
              {[480, 720, 1080, 2160].map((resolution) => {
                const resolutionItems = formData.items.filter(
                  (item) => item.quality.resolution === resolution
                )
                if (resolutionItems.length === 0) return null
                return (
                  <div key={resolution} className="p-3">
                    <div className="text-xs font-medium text-muted-foreground mb-2">
                      {resolution === 480 ? 'SD' : `${resolution}p`}
                    </div>
                    <div className="flex flex-wrap gap-x-4 gap-y-1.5">
                      {resolutionItems.map((item) => (
                        <label
                          key={item.quality.id}
                          className="flex items-center gap-2 cursor-pointer"
                        >
                          <Checkbox
                            checked={item.allowed}
                            onCheckedChange={() => toggleQuality(item.quality.id)}
                          />
                          <span className="text-sm">{item.quality.name}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>

          {/* Upgrades Toggle */}
          <div className="flex items-center justify-between rounded-lg border p-3">
            <div className="space-y-0.5">
              <Label>Upgrades</Label>
              <p className="text-xs text-muted-foreground">
                Search for better quality when file exists
              </p>
            </div>
            <Switch
              checked={formData.upgradesEnabled}
              onCheckedChange={(checked) =>
                setFormData((prev) => ({ ...prev, upgradesEnabled: checked }))
              }
            />
          </div>

          {/* Allow Auto-Approve Toggle */}
          <div className="flex items-center justify-between rounded-lg border p-3">
            <div className="space-y-0.5">
              <Label>Allow Auto-Approve</Label>
              <p className="text-xs text-muted-foreground">
                Requests using this profile can be auto-approved
              </p>
            </div>
            <Switch
              checked={formData.allowAutoApprove}
              onCheckedChange={(checked) =>
                setFormData((prev) => ({ ...prev, allowAutoApprove: checked }))
              }
            />
          </div>

          {/* Cutoff - disabled when upgrades off */}
          <div className="space-y-2">
            <Label htmlFor="cutoff" className={!formData.upgradesEnabled ? 'text-muted-foreground' : ''}>
              Cutoff
            </Label>
            <Select
              value={formData.cutoff.toString()}
              onValueChange={(v) => v && setFormData((prev) => ({ ...prev, cutoff: parseInt(v) }))}
              disabled={!formData.upgradesEnabled}
            >
              <SelectTrigger className={`w-full ${!formData.upgradesEnabled ? 'opacity-50' : ''}`}>
                {cutoffOptions.find((i) => i.quality.id === formData.cutoff)?.quality.name || 'Select cutoff'}
              </SelectTrigger>
              <SelectContent>
                {cutoffOptions.map((item) => (
                  <SelectItem key={item.quality.id} value={item.quality.id.toString()}>
                    {item.quality.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className={`text-xs ${!formData.upgradesEnabled ? 'text-muted-foreground/50' : 'text-muted-foreground'}`}>
              Stop upgrading once this quality is reached
            </p>
          </div>

          {/* Attribute Settings */}
          <div className="space-y-3">
            <h3 className="text-sm font-medium">Attribute Filters</h3>

            <AttributeSettingsSection
              label="HDR Format"
              settings={formData.hdrSettings}
              options={hdrOptions}
              disabledItems={disabledHdrItems}
              warning={attributeValidation.hdr}
              onItemModeChange={(value, mode) => updateItemMode('hdrSettings', value, mode)}
            />

            <AttributeSettingsSection
              label="Video Codec"
              settings={formData.videoCodecSettings}
              options={attributeOptions?.videoCodecs || []}
              warning={attributeValidation.videoCodec}
              onItemModeChange={(value, mode) => updateItemMode('videoCodecSettings', value, mode)}
            />

            <AttributeSettingsSection
              label="Audio Codec"
              settings={formData.audioCodecSettings}
              options={attributeOptions?.audioCodecs || []}
              warning={attributeValidation.audioCodec}
              onItemModeChange={(value, mode) => updateItemMode('audioCodecSettings', value, mode)}
            />

            <AttributeSettingsSection
              label="Audio Channels"
              settings={formData.audioChannelSettings}
              options={attributeOptions?.audioChannels || []}
              warning={attributeValidation.audioChannels}
              onItemModeChange={(value, mode) => updateItemMode('audioChannelSettings', value, mode)}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isPending || hasAttributeValidationError}>
            {isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
            {isEditing ? 'Save' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface AttributeSettingsSectionProps {
  label: string
  settings: AttributeSettings
  options: string[]
  disabledItems?: string[]
  warning?: string | null
  onItemModeChange: (value: string, mode: AttributeMode) => void
}

function AttributeSettingsSection({
  label,
  settings,
  options,
  disabledItems = [],
  warning,
  onItemModeChange,
}: AttributeSettingsSectionProps) {
  const [isOpen, setIsOpen] = useState(false)

  const getItemMode = (value: string): AttributeMode => {
    return settings.items[value] || 'acceptable'
  }

  const countByMode = (mode: AttributeMode): number => {
    return Object.values(settings.items).filter((m) => m === mode).length
  }

  const requiredCount = countByMode('required')
  const preferredCount = countByMode('preferred')
  const notAllowedCount = countByMode('notAllowed')
  const hasSettings = requiredCount > 0 || preferredCount > 0 || notAllowedCount > 0

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen} className={`border rounded-lg ${warning ? 'border-yellow-500' : ''}`}>
      <CollapsibleTrigger className="flex items-center justify-between w-full p-3 hover:bg-muted/50 transition-colors">
        <div className="flex items-center gap-2">
          {warning && <AlertTriangle className="size-4 text-yellow-500" />}
          <span className="font-medium text-sm">{label}</span>
        </div>
        <div className="flex items-center gap-2">
          {requiredCount > 0 && (
            <Badge variant="destructive" className="text-xs px-1.5 py-0">
              {requiredCount} required
            </Badge>
          )}
          {preferredCount > 0 && (
            <Badge variant="secondary" className="text-xs px-1.5 py-0">
              {preferredCount} preferred
            </Badge>
          )}
          {notAllowedCount > 0 && (
            <Badge variant="outline" className="text-xs px-1.5 py-0">
              {notAllowedCount} blocked
            </Badge>
          )}
          {!hasSettings && (
            <span className="text-muted-foreground text-xs">Acceptable</span>
          )}
          <ChevronDown
            className={`size-4 text-muted-foreground transition-transform ${isOpen ? 'rotate-180' : ''}`}
          />
        </div>
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="px-3 pb-3 space-y-1.5 border-t pt-3">
          {options.map((value) => (
            <AttributeItemRow
              key={value}
              value={value}
              mode={getItemMode(value)}
              disabled={disabledItems.includes(value)}
              onModeChange={(mode) => onItemModeChange(value, mode)}
            />
          ))}
          {warning && (
            <div className="flex items-center gap-2 mt-2 p-2 bg-yellow-500/10 border border-yellow-500/20 rounded text-yellow-600 dark:text-yellow-500">
              <AlertTriangle className="size-4 shrink-0" />
              <span className="text-xs">{warning}</span>
            </div>
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

interface AttributeItemRowProps {
  value: string
  mode: AttributeMode
  disabled?: boolean
  onModeChange: (mode: AttributeMode) => void
}

function AttributeItemRow({ value, mode, disabled, onModeChange }: AttributeItemRowProps) {
  return (
    <div className={`flex items-center justify-between py-1 ${disabled ? 'opacity-50' : ''}`}>
      <span className="text-sm">{value}</span>
      <Select
        value={mode}
        onValueChange={(v) => v && onModeChange(v as AttributeMode)}
        disabled={disabled}
      >
        <SelectTrigger className="w-28 h-7 text-xs">
          {MODE_LABELS[mode]}
        </SelectTrigger>
        <SelectContent>
          {MODE_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

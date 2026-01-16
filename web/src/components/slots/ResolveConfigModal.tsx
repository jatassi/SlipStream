import { useState, useEffect } from 'react'
import { Loader2, ChevronDown, AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
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
  useQualityProfiles,
  useUpdateQualityProfile,
  useQualityProfileAttributes,
  useSlots,
} from '@/hooks'
import type {
  CreateQualityProfileInput,
  AttributeSettings,
  AttributeMode,
  SlotConflict,
} from '@/types'
import { DEFAULT_ATTRIBUTE_SETTINGS } from '@/types'

interface ResolveConfigModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  conflicts: SlotConflict[]
  onResolved: () => void
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

const HDR_FORMATS = ['DV', 'HDR10+', 'HDR10', 'HDR', 'HLG']

export function ResolveConfigModal({
  open,
  onOpenChange,
  conflicts,
  onResolved,
}: ResolveConfigModalProps) {
  const { data: profiles } = useQualityProfiles()
  const { data: slots } = useSlots()
  const { data: attributeOptions } = useQualityProfileAttributes()
  const updateMutation = useUpdateQualityProfile()

  // Track form data for each profile being edited
  const [profileForms, setProfileForms] = useState<Record<number, CreateQualityProfileInput>>({})
  const [saving, setSaving] = useState(false)

  // Get unique profile names that have conflicts
  const conflictingProfileNames = new Set<string>()
  conflicts.forEach((c) => {
    conflictingProfileNames.add(c.slotAName)
    conflictingProfileNames.add(c.slotBName)
  })

  // Get conflicting attributes for highlighting
  const conflictingAttributes = new Set<string>()
  conflicts.forEach((c) => {
    c.issues.forEach((issue) => {
      conflictingAttributes.add(issue.attribute)
    })
  })

  // Find profiles that are assigned to slots and have conflicts
  const profilesToEdit = (profiles || []).filter((profile) => {
    const slot = slots?.find((s) => s.qualityProfileId === profile.id && s.enabled)
    if (!slot) return false
    return conflictingProfileNames.has(slot.name)
  })

  // Initialize form data when modal opens
  useEffect(() => {
    if (open && profilesToEdit.length > 0) {
      const forms: Record<number, CreateQualityProfileInput> = {}
      profilesToEdit.forEach((profile) => {
        forms[profile.id] = {
          name: profile.name,
          cutoff: profile.cutoff,
          items: profile.items,
          upgradesEnabled: profile.upgradesEnabled ?? true,
          hdrSettings: profile.hdrSettings || { ...DEFAULT_ATTRIBUTE_SETTINGS },
          videoCodecSettings: profile.videoCodecSettings || { ...DEFAULT_ATTRIBUTE_SETTINGS },
          audioCodecSettings: profile.audioCodecSettings || { ...DEFAULT_ATTRIBUTE_SETTINGS },
          audioChannelSettings: profile.audioChannelSettings || { ...DEFAULT_ATTRIBUTE_SETTINGS },
        }
      })
      setProfileForms(forms)
    }
  }, [open, profiles, slots])

  const updateProfileForm = (
    profileId: number,
    field: keyof CreateQualityProfileInput,
    value: unknown
  ) => {
    setProfileForms((prev) => ({
      ...prev,
      [profileId]: {
        ...prev[profileId],
        [field]: value,
      },
    }))
  }

  const updateItemMode = (
    profileId: number,
    settingsField: 'hdrSettings' | 'videoCodecSettings' | 'audioCodecSettings' | 'audioChannelSettings',
    value: string,
    mode: AttributeMode
  ) => {
    setProfileForms((prev) => {
      const currentForm = prev[profileId]
      if (!currentForm) return prev

      const currentItems = { ...currentForm[settingsField].items }
      if (mode === 'acceptable') {
        delete currentItems[value]
      } else {
        currentItems[value] = mode
      }

      // Auto-set mutual exclusivity for HDR/SDR
      if (settingsField === 'hdrSettings' && mode === 'required') {
        if (value === 'SDR') {
          for (const hdrFormat of HDR_FORMATS) {
            currentItems[hdrFormat] = 'notAllowed'
          }
        } else if (HDR_FORMATS.includes(value)) {
          currentItems['SDR'] = 'notAllowed'
        }
      }

      return {
        ...prev,
        [profileId]: {
          ...currentForm,
          [settingsField]: { items: currentItems },
        },
      }
    })
  }

  const toggleQuality = (profileId: number, qualityId: number) => {
    setProfileForms((prev) => {
      const currentForm = prev[profileId]
      if (!currentForm) return prev

      return {
        ...prev,
        [profileId]: {
          ...currentForm,
          items: currentForm.items.map((item) =>
            item.quality.id === qualityId ? { ...item, allowed: !item.allowed } : item
          ),
        },
      }
    })
  }

  const handleSaveAll = async () => {
    setSaving(true)
    try {
      // Save all modified profiles
      for (const profile of profilesToEdit) {
        const formData = profileForms[profile.id]
        if (formData) {
          await updateMutation.mutateAsync({
            id: profile.id,
            data: formData,
          })
        }
      }
      toast.success('All profiles updated')
      onOpenChange(false)
      onResolved()
    } catch {
      toast.error('Failed to update profiles')
    } finally {
      setSaving(false)
    }
  }

  const hdrOptions = ['SDR', ...(attributeOptions?.hdrFormats || []).filter((f) => f !== 'SDR')]

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-6xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Resolve Profile Conflicts</DialogTitle>
          <DialogDescription>
            Edit the conflicting profiles to make them mutually exclusive. Conflicting attributes are highlighted in orange.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-6" style={{ gridTemplateColumns: `repeat(${Math.min(profilesToEdit.length, 3)}, 1fr)` }}>
          {profilesToEdit.slice(0, 3).map((profile) => {
            const formData = profileForms[profile.id]
            if (!formData) return null

            const slot = slots?.find((s) => s.qualityProfileId === profile.id)
            const allowedQualities = formData.items.filter((i) => i.allowed)
            const cutoffOptions = allowedQualities.length > 0 ? allowedQualities : formData.items

            return (
              <div key={profile.id} className="space-y-4 border rounded-lg p-4">
                <div className="flex items-center gap-2">
                  <Badge variant="outline">{slot?.name || 'Unknown Slot'}</Badge>
                  <span className="font-medium">{profile.name}</span>
                </div>

                {/* Name */}
                <div className="space-y-2">
                  <Label>Name</Label>
                  <Input
                    value={formData.name}
                    onChange={(e) => updateProfileForm(profile.id, 'name', e.target.value)}
                  />
                </div>

                {/* Allowed Qualities */}
                <div className="space-y-2">
                  <Label>Allowed Qualities</Label>
                  <div className="border rounded-lg bg-muted/30 divide-y max-h-40 overflow-y-auto">
                    {[480, 720, 1080, 2160].map((resolution) => {
                      const resolutionItems = formData.items.filter(
                        (item) => item.quality.resolution === resolution
                      )
                      if (resolutionItems.length === 0) return null
                      return (
                        <div key={resolution} className="p-2">
                          <div className="text-xs font-medium text-muted-foreground mb-1">
                            {resolution === 480 ? 'SD' : `${resolution}p`}
                          </div>
                          <div className="flex flex-wrap gap-x-3 gap-y-1">
                            {resolutionItems.map((item) => (
                              <label key={item.quality.id} className="flex items-center gap-1.5 cursor-pointer">
                                <Checkbox
                                  checked={item.allowed}
                                  onCheckedChange={() => toggleQuality(profile.id, item.quality.id)}
                                />
                                <span className="text-xs">{item.quality.name}</span>
                              </label>
                            ))}
                          </div>
                        </div>
                      )
                    })}
                  </div>
                </div>

                {/* Cutoff */}
                <div className="space-y-2">
                  <Label>Cutoff</Label>
                  <Select
                    value={formData.cutoff.toString()}
                    onValueChange={(v) => v && updateProfileForm(profile.id, 'cutoff', parseInt(v))}
                  >
                    <SelectTrigger className="h-8 text-sm">
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
                </div>

                {/* Attribute Filters */}
                <div className="space-y-2">
                  <Label>Attribute Filters</Label>

                  <CompactAttributeSection
                    label="HDR Format"
                    settings={formData.hdrSettings}
                    options={hdrOptions}
                    isConflicting={conflictingAttributes.has('HDR')}
                    onItemModeChange={(value, mode) => updateItemMode(profile.id, 'hdrSettings', value, mode)}
                  />

                  <CompactAttributeSection
                    label="Video Codec"
                    settings={formData.videoCodecSettings}
                    options={attributeOptions?.videoCodecs || []}
                    isConflicting={conflictingAttributes.has('Video Codec')}
                    onItemModeChange={(value, mode) => updateItemMode(profile.id, 'videoCodecSettings', value, mode)}
                  />

                  <CompactAttributeSection
                    label="Audio Codec"
                    settings={formData.audioCodecSettings}
                    options={attributeOptions?.audioCodecs || []}
                    isConflicting={conflictingAttributes.has('Audio Codec')}
                    onItemModeChange={(value, mode) => updateItemMode(profile.id, 'audioCodecSettings', value, mode)}
                  />

                  <CompactAttributeSection
                    label="Audio Channels"
                    settings={formData.audioChannelSettings}
                    options={attributeOptions?.audioChannels || []}
                    isConflicting={conflictingAttributes.has('Audio Channels')}
                    onItemModeChange={(value, mode) => updateItemMode(profile.id, 'audioChannelSettings', value, mode)}
                  />
                </div>
              </div>
            )
          })}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSaveAll} disabled={saving}>
            {saving && <Loader2 className="size-4 mr-2 animate-spin" />}
            Save All
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface CompactAttributeSectionProps {
  label: string
  settings: AttributeSettings
  options: string[]
  isConflicting: boolean
  onItemModeChange: (value: string, mode: AttributeMode) => void
}

function CompactAttributeSection({
  label,
  settings,
  options,
  isConflicting,
  onItemModeChange,
}: CompactAttributeSectionProps) {
  const [isOpen, setIsOpen] = useState(isConflicting)

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

  const borderClass = isConflicting ? 'border-orange-400 dark:border-orange-500' : ''
  const textClass = isConflicting ? 'text-orange-600 dark:text-orange-400' : ''

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen} className={`border rounded-lg ${borderClass}`}>
      <CollapsibleTrigger className="flex items-center justify-between w-full p-2 hover:bg-muted/50 transition-colors">
        <div className="flex items-center gap-1.5">
          {isConflicting && <AlertTriangle className="size-3.5 text-orange-500" />}
          <span className={`font-medium text-xs ${textClass}`}>{label}</span>
        </div>
        <div className="flex items-center gap-1.5">
          {requiredCount > 0 && (
            <Badge variant="destructive" className="text-[10px] px-1 py-0 h-4">
              {requiredCount}
            </Badge>
          )}
          {preferredCount > 0 && (
            <Badge variant="secondary" className="text-[10px] px-1 py-0 h-4">
              {preferredCount}
            </Badge>
          )}
          {notAllowedCount > 0 && (
            <Badge variant="outline" className="text-[10px] px-1 py-0 h-4">
              {notAllowedCount}
            </Badge>
          )}
          {!hasSettings && <span className="text-muted-foreground text-[10px]">Acceptable</span>}
          <ChevronDown className={`size-3 text-muted-foreground transition-transform ${isOpen ? 'rotate-180' : ''}`} />
        </div>
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="px-2 pb-2 space-y-1 border-t pt-2">
          {options.map((value) => (
            <div key={value} className="flex items-center justify-between py-0.5">
              <span className="text-xs">{value}</span>
              <Select
                value={getItemMode(value)}
                onValueChange={(v) => v && onItemModeChange(value, v as AttributeMode)}
              >
                <SelectTrigger className="w-24 h-6 text-[10px]">
                  {MODE_LABELS[getItemMode(value)]}
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
          ))}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

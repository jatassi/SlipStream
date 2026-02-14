import { useState } from 'react'

import { AlertTriangle, ArrowRight, Check, ChevronDown, Loader2, TrendingUp, X } from 'lucide-react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  useCreateQualityProfile,
  useQualityProfileAttributes,
  useUpdateQualityProfile,
} from '@/hooks'
import type {
  AttributeMode,
  AttributeSettings,
  CreateQualityProfileInput,
  Quality,
  QualityItem,
  QualityProfile,
  UpgradeStrategy,
} from '@/types'
import { DEFAULT_ATTRIBUTE_SETTINGS, PREDEFINED_QUALITIES } from '@/types'

type QualityProfileDialogProps = {
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

const UPGRADE_STRATEGY_OPTIONS: { value: UpgradeStrategy; label: string; description: string }[] = [
  {
    value: 'balanced',
    label: 'Balanced',
    description: 'Upgrade for better resolution or source type',
  },
  {
    value: 'aggressive',
    label: 'Aggressive',
    description: 'Upgrade for any higher quality weight',
  },
  {
    value: 'resolution_only',
    label: 'Resolution Only',
    description: 'Only upgrade for higher resolution',
  },
]

const DISC_SOURCES = new Set(['bluray', 'remux'])

type UpgradeScenario = {
  from: Quality
  to: Quality
  allowed: boolean
  reason: string
}

function isUpgradeByStrategy(
  current: Quality,
  candidate: Quality,
  strategy: UpgradeStrategy,
): boolean {
  switch (strategy) {
    case 'resolution_only': {
      return candidate.resolution > current.resolution
    }
    case 'balanced': {
      if (candidate.resolution > current.resolution) {
        return true
      }
      if (candidate.resolution === current.resolution) {
        return DISC_SOURCES.has(candidate.source) && !DISC_SOURCES.has(current.source)
      }
      return false
    }
    default: {
      // aggressive
      return candidate.weight > current.weight
    }
  }
}

function generateScenarios(
  allowedItems: QualityItem[],
  strategy: UpgradeStrategy,
  cutoffId: number,
  cutoffOverridesStrategy: boolean,
): UpgradeScenario[] {
  const allowed = allowedItems.filter((i) => i.allowed).map((i) => i.quality)
  if (allowed.length < 2) {
    return []
  }

  const sorted = allowed.toSorted((a, b) => a.weight - b.weight)
  const cutoffQ = sorted.find((q) => q.id === cutoffId)
  const cutoffWeight = cutoffQ?.weight ?? (sorted.at(-1)?.weight ?? 0)

  const scenarios: UpgradeScenario[] = []
  const addedKeys = new Set<string>()
  const add = (s: UpgradeScenario) => {
    const key = `${s.from.id}-${s.to.id}`
    if (addedKeys.has(key)) {
      return
    }
    addedKeys.add(key)
    scenarios.push(s)
  }

  const resolutions = [...new Set(sorted.map((q) => q.resolution))].toSorted((a, b) => a - b)

  for (const res of resolutions) {
    const atRes = sorted.filter((q) => q.resolution === res)
    const belowCutoffAtRes = atRes.filter((q) => q.weight < cutoffWeight)
    if (belowCutoffAtRes.length === 0) {
      continue
    }

    const nonDisc = belowCutoffAtRes.filter((q) => !DISC_SOURCES.has(q.source))
    const disc = atRes.filter((q) => DISC_SOURCES.has(q.source))

    // Within non-disc (e.g. WEBRip → WEBDL): aggressive ✓, balanced ✗, res_only ✗
    if (nonDisc.length >= 2) {
      const from = nonDisc[0]
      const to = nonDisc.at(-1)
      if (to) {
        const passes = isUpgradeByStrategy(from, to, strategy)
        add({ from, to, allowed: passes, reason: passes ? 'Better source' : 'Same tier' })
      }
    }

    // Non-disc → disc (e.g. WEBDL → Bluray): aggressive ✓, balanced ✓, res_only ✗
    if (nonDisc.length > 0 && disc.length > 0) {
      const from = nonDisc.at(-1)
      const to = disc[0]
      if (from) {
        const passes = isUpgradeByStrategy(from, to, strategy)
        add({ from, to, allowed: passes, reason: passes ? 'Non-disc to disc' : 'Same resolution' })
      }
    }

    // Within disc (e.g. Bluray → Remux): aggressive ✓, balanced ✗, res_only ✗
    if (disc.length >= 2 && disc.some((q) => q.weight < cutoffWeight)) {
      const from = disc[0]
      const to = disc.at(-1)
      if (from.weight < cutoffWeight && to) {
        const passes = isUpgradeByStrategy(from, to, strategy)
        add({ from, to, allowed: passes, reason: passes ? 'Better source' : 'Same tier' })
      }
    }
  }

  // Resolution upgrades across tiers
  for (let i = 0; i < resolutions.length - 1; i++) {
    const fromQ = sorted.filter((q) => q.resolution === resolutions[i])
    const toQ = sorted.find((q) => q.resolution === resolutions[i + 1])
    const from = fromQ.at(-1)
    const to = toQ
    if (!from || !to || from.weight >= cutoffWeight) {
      continue
    }
    add({ from, to, allowed: true, reason: 'Higher resolution' })
  }

  // Cutoff override: show a scenario where the override bypasses strategy
  if (cutoffOverridesStrategy && cutoffQ) {
    const belowCutoff = sorted.filter((q) => q.weight < cutoffWeight)
    const overrideFrom = [...belowCutoff]
      .reverse()
      .find((q) => !isUpgradeByStrategy(q, cutoffQ, strategy))
    if (overrideFrom) {
      add({ from: overrideFrom, to: cutoffQ, allowed: true, reason: 'Cutoff override' })
    }
  }

  // Cutoff block
  const atCutoff = sorted.filter((q) => q.weight >= cutoffWeight)
  if (atCutoff.length > 0) {
    const from = atCutoff[0]
    const higher = sorted.find((q) => q.weight > from.weight)
    if (higher) {
      add({ from, to: higher, allowed: false, reason: 'At cutoff' })
    }
  }

  // Sort: allowed first, then blocked
  return scenarios.toSorted((a, b) => (a.allowed === b.allowed ? 0 : a.allowed ? -1 : 1))
}

const defaultItems: QualityItem[] = PREDEFINED_QUALITIES.map((q) => ({
  quality: q,
  allowed: q.weight >= 10,
}))

const defaultFormData: CreateQualityProfileInput = {
  name: '',
  cutoff: 10,
  upgradesEnabled: true,
  upgradeStrategy: 'balanced',
  cutoffOverridesStrategy: false,
  allowAutoApprove: false,
  items: defaultItems,
  hdrSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
  videoCodecSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
  audioCodecSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
  audioChannelSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
}

const validateAttributeGroup = (
  settings: AttributeSettings,
  options: string[],
): string | null => {
  if (options.length === 0) {
    return null
  }

  const modes = options.map((opt) => settings.items[opt] ?? 'acceptable')
  const nonAcceptableModes = modes.filter((m) => m !== 'acceptable')

  // If not all items have a non-acceptable mode set, it's valid
  if (nonAcceptableModes.length !== options.length) {
    return null
  }

  // Check if all are the same mode
  const firstMode = nonAcceptableModes[0]
  const allSame = nonAcceptableModes.every((m) => m === firstMode)

  if (allSame) {
    switch (firstMode) {
      case 'required': {
        return 'All items set to Required - no release can match all requirements'
      }
      case 'preferred': {
        return 'All items set to Preferred - this is equivalent to Acceptable'
      }
      case 'notAllowed': {
        return 'All items set to Not Allowed - no release can match'
      }
    }
  }
  return null
}

export function QualityProfileDialog({ open, onOpenChange, profile }: QualityProfileDialogProps) {
  const [formData, setFormData] = useState<CreateQualityProfileInput>(defaultFormData)
  const [prevOpen, setPrevOpen] = useState(open)
  const [prevProfile, setPrevProfile] = useState(profile)

  const createMutation = useCreateQualityProfile()
  const updateMutation = useUpdateQualityProfile()
  const { data: attributeOptions } = useQualityProfileAttributes()

  const isEditing = !!profile

  // Reset form when dialog opens or profile changes (React-recommended pattern)
  if (open !== prevOpen || profile !== prevProfile) {
    setPrevOpen(open)
    setPrevProfile(profile)
    if (open) {
      if (profile) {
        setFormData({
          name: profile.name,
          cutoff: profile.cutoff,
          upgradesEnabled: profile.upgradesEnabled,
          upgradeStrategy: profile.upgradeStrategy,
          cutoffOverridesStrategy: profile.cutoffOverridesStrategy,
          allowAutoApprove: profile.allowAutoApprove,
          items: profile.items,
          hdrSettings: profile.hdrSettings,
          videoCodecSettings: profile.videoCodecSettings,
          audioCodecSettings: profile.audioCodecSettings,
          audioChannelSettings: profile.audioChannelSettings,
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
  }

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
      if (isEditing) {
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
        item.quality.id === qualityId ? { ...item, allowed: !item.allowed } : item,
      ),
    }))
  }

  const HDR_FORMATS = ['DV', 'HDR10+', 'HDR10', 'HDR', 'HLG']

  const updateItemMode = (
    field: 'hdrSettings' | 'videoCodecSettings' | 'audioCodecSettings' | 'audioChannelSettings',
    value: string,
    mode: AttributeMode,
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
          currentItems.SDR = 'notAllowed'
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
    if (hdrItems.SDR === 'required') {
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

  const hdrOptions = ['SDR', ...(attributeOptions?.hdrFormats ?? []).filter((f) => f !== 'SDR')]
  const attributeValidation = {
    hdr: validateAttributeGroup(formData.hdrSettings, hdrOptions),
    videoCodec: validateAttributeGroup(
      formData.videoCodecSettings,
      attributeOptions?.videoCodecs || [],
    ),
    audioCodec: validateAttributeGroup(
      formData.audioCodecSettings,
      attributeOptions?.audioCodecs || [],
    ),
    audioChannels: validateAttributeGroup(
      formData.audioChannelSettings,
      attributeOptions?.audioChannels || [],
    ),
  }

  const hasAttributeValidationError = Object.values(attributeValidation).some((v) => v !== null)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl">
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
            <div className="bg-muted/30 divide-y rounded-lg border">
              {[480, 720, 1080, 2160].map((resolution) => {
                const resolutionItems = formData.items.filter(
                  (item) => item.quality.resolution === resolution,
                )
                if (resolutionItems.length === 0) {
                  return null
                }
                return (
                  <div key={resolution} className="p-3">
                    <div className="text-muted-foreground mb-2 text-xs font-medium">
                      {resolution === 480 ? 'SD' : `${resolution}p`}
                    </div>
                    <div className="flex flex-wrap gap-x-4 gap-y-1.5">
                      {resolutionItems.map((item) => (
                        <label
                          key={item.quality.id}
                          className="flex cursor-pointer items-center gap-2"
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
              <p className="text-muted-foreground text-xs">
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

          {/* Upgrade Strategy */}
          <div className="space-y-2">
            <Label className={formData.upgradesEnabled ? '' : 'text-muted-foreground'}>
              Upgrade Strategy
            </Label>
            <Select
              value={formData.upgradeStrategy}
              onValueChange={(v) =>
                v && setFormData((prev) => ({ ...prev, upgradeStrategy: v as UpgradeStrategy }))
              }
              disabled={!formData.upgradesEnabled}
            >
              <SelectTrigger className={`w-full ${formData.upgradesEnabled ? '' : 'opacity-50'}`}>
                {UPGRADE_STRATEGY_OPTIONS.find((o) => o.value === formData.upgradeStrategy)
                  ?.label || 'Select strategy'}
              </SelectTrigger>
              <SelectContent>
                {UPGRADE_STRATEGY_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    <div>
                      <div>{option.label}</div>
                      <div className="text-muted-foreground text-xs">{option.description}</div>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Upgrade Strategy Preview */}
          {formData.upgradesEnabled && allowedQualities.length >= 2 ? (
            <UpgradeStrategyPreview
              allowedQualities={formData.items}
              strategy={formData.upgradeStrategy}
              cutoffId={formData.cutoff}
              cutoffOverridesStrategy={formData.cutoffOverridesStrategy}
            />
          ) : null}

          {/* Cutoff Overrides Strategy Toggle */}
          <div
            className={`flex items-center justify-between rounded-lg border p-3 ${
              !formData.upgradesEnabled || formData.upgradeStrategy === 'aggressive'
                ? 'opacity-50'
                : ''
            }`}
          >
            <div className="space-y-0.5">
              <Label
                className={
                  !formData.upgradesEnabled || formData.upgradeStrategy === 'aggressive'
                    ? 'text-muted-foreground'
                    : ''
                }
              >
                Cutoff Overrides Strategy
              </Label>
              <p className="text-muted-foreground text-xs">
                Always grab cutoff quality even if strategy would block it
              </p>
            </div>
            <Switch
              checked={formData.cutoffOverridesStrategy}
              disabled={!formData.upgradesEnabled || formData.upgradeStrategy === 'aggressive'}
              onCheckedChange={(checked) =>
                setFormData((prev) => ({ ...prev, cutoffOverridesStrategy: checked }))
              }
            />
          </div>

          {/* Allow Auto-Approve Toggle */}
          <div className="flex items-center justify-between rounded-lg border p-3">
            <div className="space-y-0.5">
              <Label>Allow Auto-Approve</Label>
              <p className="text-muted-foreground text-xs">
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
            <Label
              htmlFor="cutoff"
              className={formData.upgradesEnabled ? '' : 'text-muted-foreground'}
            >
              Cutoff
            </Label>
            <Select
              value={formData.cutoff.toString()}
              onValueChange={(v) =>
                v && setFormData((prev) => ({ ...prev, cutoff: Number.parseInt(v) }))
              }
              disabled={!formData.upgradesEnabled}
            >
              <SelectTrigger className={`w-full ${formData.upgradesEnabled ? '' : 'opacity-50'}`}>
                {cutoffOptions.find((i) => i.quality.id === formData.cutoff)?.quality.name ||
                  'Select cutoff'}
              </SelectTrigger>
              <SelectContent>
                {cutoffOptions.map((item) => (
                  <SelectItem key={item.quality.id} value={item.quality.id.toString()}>
                    {item.quality.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p
              className={`text-xs ${formData.upgradesEnabled ? 'text-muted-foreground' : 'text-muted-foreground/50'}`}
            >
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
              onItemModeChange={(value, mode) =>
                updateItemMode('audioChannelSettings', value, mode)
              }
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isPending || hasAttributeValidationError}>
            {isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            {isEditing ? 'Save' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function ScenarioRow({ scenario }: { scenario: UpgradeScenario }) {
  return (
    <div className="flex items-center gap-2 text-sm">
      {scenario.allowed ? (
        <Check className="size-3.5 shrink-0 text-green-500" />
      ) : (
        <X className="size-3.5 shrink-0 text-red-500" />
      )}
      <Badge
        variant="secondary"
        className={`px-1.5 py-0 text-xs font-normal ${scenario.allowed ? '' : 'opacity-60'}`}
      >
        {scenario.from.name}
        <ArrowRight className="mx-1 inline size-3" />
        {scenario.to.name}
      </Badge>
      <span className="text-muted-foreground text-[10px]">{scenario.reason}</span>
    </div>
  )
}

function UpgradeStrategyPreview({
  allowedQualities,
  strategy,
  cutoffId,
  cutoffOverridesStrategy,
}: {
  allowedQualities: QualityItem[]
  strategy: UpgradeStrategy
  cutoffId: number
  cutoffOverridesStrategy: boolean
}) {
  const [isOpen, setIsOpen] = useState(false)
  const scenarios = generateScenarios(allowedQualities, strategy, cutoffId, cutoffOverridesStrategy)
  if (scenarios.length === 0) {
    return null
  }

  const allowedCount = scenarios.filter((s) => s.allowed).length
  const blockedCount = scenarios.length - allowedCount

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <div className="border-border/60 bg-muted/20 rounded-lg border px-3 py-2.5">
        <CollapsibleTrigger className="flex w-full items-center justify-between">
          <div className="text-muted-foreground flex items-center gap-1.5 text-xs font-medium">
            <TrendingUp className="size-3" />
            Upgrade Preview
          </div>
          <div className="flex items-center gap-2">
            {allowedCount > 0 && (
              <span className="text-[10px] text-green-500">{allowedCount} allowed</span>
            )}
            {blockedCount > 0 && (
              <span className="text-[10px] text-red-500">{blockedCount} blocked</span>
            )}
            <ChevronDown
              className={`text-muted-foreground size-3.5 transition-transform ${isOpen ? 'rotate-180' : ''}`}
            />
          </div>
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-1.5 pt-1.5">
          {scenarios.map((s) => (
            <ScenarioRow key={`${s.from.id}-${s.to.id}-${s.reason}`} scenario={s} />
          ))}
        </CollapsibleContent>
      </div>
    </Collapsible>
  )
}

type AttributeSettingsSectionProps = {
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
    return settings.items[value] ?? 'acceptable'
  }

  const countByMode = (mode: AttributeMode): number => {
    return Object.values(settings.items).filter((m) => m === mode).length
  }

  const requiredCount = countByMode('required')
  const preferredCount = countByMode('preferred')
  const notAllowedCount = countByMode('notAllowed')
  const hasSettings = requiredCount > 0 || preferredCount > 0 || notAllowedCount > 0

  return (
    <Collapsible
      open={isOpen}
      onOpenChange={setIsOpen}
      className={`rounded-lg border ${warning ? 'border-yellow-500' : ''}`}
    >
      <CollapsibleTrigger className="hover:bg-muted/50 flex w-full items-center justify-between p-3 transition-colors">
        <div className="flex items-center gap-2">
          {warning ? <AlertTriangle className="size-4 text-yellow-500" /> : null}
          <span className="text-sm font-medium">{label}</span>
        </div>
        <div className="flex items-center gap-2">
          {requiredCount > 0 && (
            <Badge variant="destructive" className="px-1.5 py-0 text-xs">
              {requiredCount} required
            </Badge>
          )}
          {preferredCount > 0 && (
            <Badge variant="secondary" className="px-1.5 py-0 text-xs">
              {preferredCount} preferred
            </Badge>
          )}
          {notAllowedCount > 0 && (
            <Badge variant="outline" className="px-1.5 py-0 text-xs">
              {notAllowedCount} blocked
            </Badge>
          )}
          {!hasSettings && <span className="text-muted-foreground text-xs">Acceptable</span>}
          <ChevronDown
            className={`text-muted-foreground size-4 transition-transform ${isOpen ? 'rotate-180' : ''}`}
          />
        </div>
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="space-y-1.5 border-t px-3 pt-3 pb-3">
          {options.map((value) => (
            <AttributeItemRow
              key={value}
              value={value}
              mode={getItemMode(value)}
              disabled={disabledItems.includes(value)}
              onModeChange={(mode) => onItemModeChange(value, mode)}
            />
          ))}
          {warning ? (
            <div className="mt-2 flex items-center gap-2 rounded border border-yellow-500/20 bg-yellow-500/10 p-2 text-yellow-600 dark:text-yellow-500">
              <AlertTriangle className="size-4 shrink-0" />
              <span className="text-xs">{warning}</span>
            </div>
          ) : null}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

type AttributeItemRowProps = {
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
        <SelectTrigger className="h-7 w-28 text-xs">{MODE_LABELS[mode]}</SelectTrigger>
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

import { useState } from 'react'
import { Loader2, ChevronDown, AlertTriangle, TrendingUp, ArrowRight, Check, X } from 'lucide-react'
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
  Quality,
  QualityProfile,
  QualityItem,
  CreateQualityProfileInput,
  AttributeSettings,
  AttributeMode,
  UpgradeStrategy,
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

const UPGRADE_STRATEGY_OPTIONS: { value: UpgradeStrategy; label: string; description: string }[] = [
  { value: 'balanced', label: 'Balanced', description: 'Upgrade for better resolution or source type' },
  { value: 'aggressive', label: 'Aggressive', description: 'Upgrade for any higher quality weight' },
  { value: 'resolution_only', label: 'Resolution Only', description: 'Only upgrade for higher resolution' },
]

const DISC_SOURCES = new Set(['bluray', 'remux'])

interface UpgradeScenario {
  from: Quality
  to: Quality
  allowed: boolean
  reason: string
}

function isUpgradeByStrategy(current: Quality, candidate: Quality, strategy: UpgradeStrategy): boolean {
  switch (strategy) {
    case 'resolution_only':
      return candidate.resolution > current.resolution
    case 'balanced':
      if (candidate.resolution > current.resolution) return true
      if (candidate.resolution === current.resolution) {
        return DISC_SOURCES.has(candidate.source) && !DISC_SOURCES.has(current.source)
      }
      return false
    default: // aggressive
      return candidate.weight > current.weight
  }
}

function generateScenarios(
  allowedItems: QualityItem[],
  strategy: UpgradeStrategy,
  cutoffId: number,
  cutoffOverridesStrategy: boolean
): UpgradeScenario[] {
  const allowed = allowedItems.filter((i) => i.allowed).map((i) => i.quality)
  if (allowed.length < 2) return []

  const sorted = [...allowed].sort((a, b) => a.weight - b.weight)
  const cutoffQ = sorted.find((q) => q.id === cutoffId)
  const cutoffWeight = cutoffQ?.weight ?? sorted[sorted.length - 1].weight

  const scenarios: UpgradeScenario[] = []
  const addedKeys = new Set<string>()
  const add = (s: UpgradeScenario) => {
    const key = `${s.from.id}-${s.to.id}`
    if (addedKeys.has(key)) return
    addedKeys.add(key)
    scenarios.push(s)
  }

  const resolutions = [...new Set(sorted.map((q) => q.resolution))].sort((a, b) => a - b)

  for (const res of resolutions) {
    const atRes = sorted.filter((q) => q.resolution === res)
    const belowCutoffAtRes = atRes.filter((q) => q.weight < cutoffWeight)
    if (belowCutoffAtRes.length === 0) continue

    const nonDisc = belowCutoffAtRes.filter((q) => !DISC_SOURCES.has(q.source))
    const disc = atRes.filter((q) => DISC_SOURCES.has(q.source))

    // Within non-disc (e.g. WEBRip → WEBDL): aggressive ✓, balanced ✗, res_only ✗
    if (nonDisc.length >= 2) {
      const from = nonDisc[0]
      const to = nonDisc[nonDisc.length - 1]
      const passes = isUpgradeByStrategy(from, to, strategy)
      add({ from, to, allowed: passes, reason: passes ? 'Better source' : 'Same tier' })
    }

    // Non-disc → disc (e.g. WEBDL → Bluray): aggressive ✓, balanced ✓, res_only ✗
    if (nonDisc.length > 0 && disc.length > 0) {
      const from = nonDisc[nonDisc.length - 1]
      const to = disc[0]
      const passes = isUpgradeByStrategy(from, to, strategy)
      add({ from, to, allowed: passes, reason: passes ? 'Non-disc to disc' : 'Same resolution' })
    }

    // Within disc (e.g. Bluray → Remux): aggressive ✓, balanced ✗, res_only ✗
    if (disc.length >= 2 && disc.some((q) => q.weight < cutoffWeight)) {
      const from = disc[0]
      const to = disc[disc.length - 1]
      if (from.weight < cutoffWeight) {
        const passes = isUpgradeByStrategy(from, to, strategy)
        add({ from, to, allowed: passes, reason: passes ? 'Better source' : 'Same tier' })
      }
    }
  }

  // Resolution upgrades across tiers
  for (let i = 0; i < resolutions.length - 1; i++) {
    const fromQ = sorted.filter((q) => q.resolution === resolutions[i])
    const toQ = sorted.filter((q) => q.resolution === resolutions[i + 1])
    const from = fromQ[fromQ.length - 1]
    const to = toQ[0]
    if (from.weight >= cutoffWeight) continue
    add({ from, to, allowed: true, reason: 'Higher resolution' })
  }

  // Cutoff override: show a scenario where the override bypasses strategy
  if (cutoffOverridesStrategy && cutoffQ) {
    const belowCutoff = sorted.filter((q) => q.weight < cutoffWeight)
    const overrideFrom = [...belowCutoff].reverse().find(
      (q) => !isUpgradeByStrategy(q, cutoffQ, strategy)
    )
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
  return scenarios.sort((a, b) => (a.allowed === b.allowed ? 0 : a.allowed ? -1 : 1))
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

export function QualityProfileDialog({
  open,
  onOpenChange,
  profile,
}: QualityProfileDialogProps) {
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
          upgradesEnabled: profile.upgradesEnabled ?? true,
          upgradeStrategy: profile.upgradeStrategy || 'balanced',
          cutoffOverridesStrategy: profile.cutoffOverridesStrategy ?? false,
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

          {/* Upgrade Strategy */}
          <div className="space-y-2">
            <Label className={!formData.upgradesEnabled ? 'text-muted-foreground' : ''}>
              Upgrade Strategy
            </Label>
            <Select
              value={formData.upgradeStrategy}
              onValueChange={(v) => v && setFormData((prev) => ({ ...prev, upgradeStrategy: v as UpgradeStrategy }))}
              disabled={!formData.upgradesEnabled}
            >
              <SelectTrigger className={`w-full ${!formData.upgradesEnabled ? 'opacity-50' : ''}`}>
                {UPGRADE_STRATEGY_OPTIONS.find((o) => o.value === formData.upgradeStrategy)?.label || 'Select strategy'}
              </SelectTrigger>
              <SelectContent>
                {UPGRADE_STRATEGY_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    <div>
                      <div>{option.label}</div>
                      <div className="text-xs text-muted-foreground">{option.description}</div>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Upgrade Strategy Preview */}
          {formData.upgradesEnabled && allowedQualities.length >= 2 && (
            <UpgradeStrategyPreview
              allowedQualities={formData.items}
              strategy={formData.upgradeStrategy}
              cutoffId={formData.cutoff}
              cutoffOverridesStrategy={formData.cutoffOverridesStrategy}
            />
          )}

          {/* Cutoff Overrides Strategy Toggle */}
          <div className={`flex items-center justify-between rounded-lg border p-3 ${
            !formData.upgradesEnabled || formData.upgradeStrategy === 'aggressive' ? 'opacity-50' : ''
          }`}>
            <div className="space-y-0.5">
              <Label className={!formData.upgradesEnabled || formData.upgradeStrategy === 'aggressive' ? 'text-muted-foreground' : ''}>
                Cutoff Overrides Strategy
              </Label>
              <p className="text-xs text-muted-foreground">
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
        className={`font-normal text-xs px-1.5 py-0 ${scenario.allowed ? '' : 'opacity-60'}`}
      >
        {scenario.from.name}
        <ArrowRight className="size-3 mx-1 inline" />
        {scenario.to.name}
      </Badge>
      <span className="text-[10px] text-muted-foreground">{scenario.reason}</span>
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
  if (scenarios.length === 0) return null

  const allowedCount = scenarios.filter((s) => s.allowed).length
  const blockedCount = scenarios.length - allowedCount

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <div className="rounded-lg border border-border/60 bg-muted/20 px-3 py-2.5">
        <CollapsibleTrigger className="flex items-center justify-between w-full">
          <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
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
              className={`size-3.5 text-muted-foreground transition-transform ${isOpen ? 'rotate-180' : ''}`}
            />
          </div>
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-1.5 pt-1.5">
          {scenarios.map((s, i) => (
            <ScenarioRow key={i} scenario={s} />
          ))}
        </CollapsibleContent>
      </div>
    </Collapsible>
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

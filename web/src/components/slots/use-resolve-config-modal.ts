import { useMemo, useState } from 'react'

import { toast } from 'sonner'

import {
  useQualityProfileAttributes,
  useQualityProfiles,
  useSlots,
  useUpdateQualityProfile,
} from '@/hooks'
import type {
  AttributeMode,
  CreateQualityProfileInput,
  QualityProfile,
  Slot,
  SlotConflict,
} from '@/types'

import type { AttributeSettingsField } from './resolve-config-constants'
import { HDR_FORMATS } from './resolve-config-constants'

type UseResolveConfigModalArgs = {
  open: boolean
  onOpenChange: (open: boolean) => void
  conflicts: SlotConflict[]
  onResolved: () => void
}

type ItemModeUpdate = {
  settingsField: AttributeSettingsField
  value: string
  mode: AttributeMode
}

type SaveContext = {
  profilesToEdit: QualityProfile[]
  profileForms: Record<number, CreateQualityProfileInput>
  updateMutation: ReturnType<typeof useUpdateQualityProfile>
  setSaving: (v: boolean) => void
  onOpenChange: (open: boolean) => void
  onResolved: () => void
}

export function useResolveConfigModal({
  open,
  onOpenChange,
  conflicts,
  onResolved,
}: UseResolveConfigModalArgs) {
  const { data: profiles } = useQualityProfiles()
  const { data: slots } = useSlots()
  const { data: attributeOptions } = useQualityProfileAttributes()
  const updateMutation = useUpdateQualityProfile()

  const [profileForms, setProfileForms] = useState<Record<number, CreateQualityProfileInput>>({})
  const [saving, setSaving] = useState(false)
  const [prevOpen, setPrevOpen] = useState(open)

  const conflictingAttributes = useMemo(() => collectConflictingAttributes(conflicts), [conflicts])
  const profilesToEdit = useProfilesToEdit(profiles, slots, conflicts)

  if (open !== prevOpen) {
    setPrevOpen(open)
    if (open && profilesToEdit.length > 0) {
      setProfileForms(buildInitialForms(profilesToEdit))
    }
  }

  const updateProfileForm = createFieldUpdater(setProfileForms)
  const updateItemMode = createItemModeUpdater(setProfileForms)
  const toggleQuality = createQualityToggler(setProfileForms)

  const handleSaveAll = () =>
    saveAllProfiles({ profilesToEdit, profileForms, updateMutation, setSaving, onOpenChange, onResolved })

  const hdrOptions = ['SDR', ...(attributeOptions?.hdrFormats ?? []).filter((f) => f !== 'SDR')]

  return {
    slots,
    attributeOptions,
    profileForms,
    saving,
    conflictingAttributes,
    profilesToEdit,
    hdrOptions,
    updateProfileForm,
    updateItemMode,
    toggleQuality,
    handleSaveAll,
  }
}

function createFieldUpdater(
  setForms: React.Dispatch<React.SetStateAction<Record<number, CreateQualityProfileInput>>>,
) {
  return (profileId: number, field: keyof CreateQualityProfileInput, value: unknown) => {
    setForms((prev) => ({
      ...prev,
      [profileId]: { ...prev[profileId], [field]: value },
    }))
  }
}

function createItemModeUpdater(
  setForms: React.Dispatch<React.SetStateAction<Record<number, CreateQualityProfileInput>>>,
) {
  return (profileId: number, update: ItemModeUpdate) => {
    setForms((prev) => applyItemModeUpdate(prev, profileId, update))
  }
}

function createQualityToggler(
  setForms: React.Dispatch<React.SetStateAction<Record<number, CreateQualityProfileInput>>>,
) {
  return (profileId: number, qualityId: number) => {
    setForms((prev) => applyQualityToggle(prev, profileId, qualityId))
  }
}

async function saveAllProfiles(ctx: SaveContext) {
  ctx.setSaving(true)
  try {
    for (const profile of ctx.profilesToEdit) {
      await ctx.updateMutation.mutateAsync({
        id: profile.id,
        data: ctx.profileForms[profile.id],
      })
    }
    toast.success('All profiles updated')
    ctx.onOpenChange(false)
    ctx.onResolved()
  } catch {
    toast.error('Failed to update profiles')
  } finally {
    ctx.setSaving(false)
  }
}

function isProfileConflicting(
  profile: QualityProfile,
  slots: Slot[] | undefined,
  conflictingNames: Set<string>,
): boolean {
  const slot = slots?.find((s) => s.qualityProfileId === profile.id && s.enabled)
  return slot !== undefined && conflictingNames.has(slot.name)
}

function useProfilesToEdit(
  profiles: QualityProfile[] | undefined,
  slots: Slot[] | undefined,
  conflicts: SlotConflict[],
) {
  const conflictingProfileNames = useMemo(() => {
    const names = new Set<string>()
    for (const c of conflicts) {
      names.add(c.slotAName)
      names.add(c.slotBName)
    }
    return names
  }, [conflicts])

  return useMemo(
    () =>
      (profiles ?? []).filter((profile) =>
        isProfileConflicting(profile, slots, conflictingProfileNames),
      ),
    [profiles, slots, conflictingProfileNames],
  )
}

function collectConflictingAttributes(conflicts: SlotConflict[]): Set<string> {
  const attrs = new Set<string>()
  for (const c of conflicts) {
    for (const issue of c.issues) {
      attrs.add(issue.attribute)
    }
  }
  return attrs
}

function buildInitialForms(
  profilesToEdit: QualityProfile[],
): Record<number, CreateQualityProfileInput> {
  const forms: Record<number, CreateQualityProfileInput> = {}
  for (const profile of profilesToEdit) {
    forms[profile.id] = {
      name: profile.name,
      cutoff: profile.cutoff,
      items: profile.items,
      upgradesEnabled: profile.upgradesEnabled,
      upgradeStrategy: profile.upgradeStrategy,
      cutoffOverridesStrategy: profile.cutoffOverridesStrategy,
      allowAutoApprove: profile.allowAutoApprove,
      hdrSettings: profile.hdrSettings,
      videoCodecSettings: profile.videoCodecSettings,
      audioCodecSettings: profile.audioCodecSettings,
      audioChannelSettings: profile.audioChannelSettings,
    }
  }
  return forms
}

function applyItemModeUpdate(
  prev: Record<number, CreateQualityProfileInput>,
  profileId: number,
  { settingsField, value, mode }: ItemModeUpdate,
): Record<number, CreateQualityProfileInput> {
  const currentForm = prev[profileId]
  const currentItems = { ...currentForm[settingsField].items }

  if (mode === 'acceptable') {
    const { [value]: _, ...rest } = currentItems
    return { ...prev, [profileId]: { ...currentForm, [settingsField]: { items: rest } } }
  }

  currentItems[value] = mode

  if (settingsField === 'hdrSettings' && mode === 'required') {
    applyHdrExclusivity(currentItems, value)
  }

  return { ...prev, [profileId]: { ...currentForm, [settingsField]: { items: currentItems } } }
}

function applyHdrExclusivity(items: Record<string, AttributeMode>, value: string) {
  if (value === 'SDR') {
    for (const format of HDR_FORMATS) {
      items[format] = 'notAllowed'
    }
  } else if (HDR_FORMATS.includes(value)) {
    items.SDR = 'notAllowed'
  }
}

function applyQualityToggle(
  prev: Record<number, CreateQualityProfileInput>,
  profileId: number,
  qualityId: number,
): Record<number, CreateQualityProfileInput> {
  const currentForm = prev[profileId]
  return {
    ...prev,
    [profileId]: {
      ...currentForm,
      items: currentForm.items.map((item) =>
        item.quality.id === qualityId ? { ...item, allowed: !item.allowed } : item,
      ),
    },
  }
}

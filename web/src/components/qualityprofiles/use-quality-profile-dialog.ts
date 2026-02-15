import { useState } from 'react'

import { toast } from 'sonner'

import {
  useCreateQualityProfile,
  useQualityProfileAttributes,
  useUpdateQualityProfile,
} from '@/hooks'
import type { AttributeMode, CreateQualityProfileInput, QualityProfile } from '@/types'
import { DEFAULT_ATTRIBUTE_SETTINGS } from '@/types'

import { defaultFormData, defaultItems, HDR_FORMATS } from './constants'
import { validateAttributeGroup } from './upgrade-scenarios'

type SettingsField =
  | 'hdrSettings'
  | 'videoCodecSettings'
  | 'audioCodecSettings'
  | 'audioChannelSettings'

function buildFormFromProfile(profile: QualityProfile): CreateQualityProfileInput {
  return {
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
  }
}

function buildFreshForm(): CreateQualityProfileInput {
  return {
    ...defaultFormData,
    items: defaultItems.map((i) => ({ ...i })),
    hdrSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
    videoCodecSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
    audioCodecSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
    audioChannelSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
  }
}

function applyHdrMutualExclusion(
  currentItems: Record<string, AttributeMode>,
  value: string,
  mode: AttributeMode,
): void {
  if (mode !== 'required') {
    return
  }
  if (value === 'SDR') {
    for (const hdrFormat of HDR_FORMATS) {
      currentItems[hdrFormat] = 'notAllowed'
    }
  } else if (HDR_FORMATS.includes(value)) {
    currentItems.SDR = 'notAllowed'
  }
}

function computeDisabledHdrItems(hdrItems: Record<string, AttributeMode>): string[] {
  const disabled: string[] = []
  if (hdrItems.SDR === 'required') {
    disabled.push(...HDR_FORMATS)
  }
  if (HDR_FORMATS.some((f) => hdrItems[f] === 'required')) {
    disabled.push('SDR')
  }
  return disabled
}

function computeValidation(
  formData: CreateQualityProfileInput,
  hdrOptions: string[],
  attributeOptions: { videoCodecs: string[]; audioCodecs: string[]; audioChannels: string[] } | undefined,
) {
  return {
    hdr: validateAttributeGroup(formData.hdrSettings, hdrOptions),
    videoCodec: validateAttributeGroup(
      formData.videoCodecSettings,
      attributeOptions?.videoCodecs ?? [],
    ),
    audioCodec: validateAttributeGroup(
      formData.audioCodecSettings,
      attributeOptions?.audioCodecs ?? [],
    ),
    audioChannels: validateAttributeGroup(
      formData.audioChannelSettings,
      attributeOptions?.audioChannels ?? [],
    ),
  }
}

function removeKey(
  obj: Record<string, AttributeMode>,
  key: string,
): Record<string, AttributeMode> {
  const result = Object.create(null) as Record<string, AttributeMode>
  for (const [k, v] of Object.entries(obj)) {
    if (k !== key) {
      result[k] = v
    }
  }
  return result
}

function useFormSync(
  open: boolean,
  profile: QualityProfile | null | undefined,
) {
  const [formData, setFormData] = useState<CreateQualityProfileInput>(defaultFormData)
  const [prevOpen, setPrevOpen] = useState(open)
  const [prevProfile, setPrevProfile] = useState(profile)

  if (open !== prevOpen || profile !== prevProfile) {
    setPrevOpen(open)
    setPrevProfile(profile)
    if (open) {
      setFormData(profile ? buildFormFromProfile(profile) : buildFreshForm())
    }
  }

  return { formData, setFormData }
}

export function useQualityProfileDialog(
  open: boolean,
  onOpenChange: (open: boolean) => void,
  profile?: QualityProfile | null,
) {
  const { formData, setFormData } = useFormSync(open, profile)
  const createMutation = useCreateQualityProfile()
  const updateMutation = useUpdateQualityProfile()
  const { data: attributeOptions } = useQualityProfileAttributes()
  const isEditing = !!profile

  const handleSubmit = async () => {
    await submitForm({ formData, isEditing, profile, createMutation, updateMutation, onOpenChange })
  }

  const toggleQuality = (qualityId: number) => {
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) =>
        item.quality.id === qualityId ? { ...item, allowed: !item.allowed } : item,
      ),
    }))
  }

  const updateItemMode = (field: SettingsField, value: string, mode: AttributeMode) => {
    setFormData((prev) => buildUpdatedSettings(prev, { field, value, mode }))
  }

  const updateField = (key: string, value: unknown) => {
    setFormData((prev) => ({ ...prev, [key]: value }))
  }

  const hdrOptions = ['SDR', ...(attributeOptions?.hdrFormats ?? []).filter((f) => f !== 'SDR')]
  const allowedQualities = formData.items.filter((i) => i.allowed)
  const attributeValidation = computeValidation(formData, hdrOptions, attributeOptions)

  return {
    formData,
    isEditing,
    isPending: createMutation.isPending || updateMutation.isPending,
    allowedQualities,
    cutoffOptions: allowedQualities.length > 0 ? allowedQualities : formData.items,
    disabledHdrItems: computeDisabledHdrItems(formData.hdrSettings.items),
    hdrOptions,
    attributeOptions,
    attributeValidation,
    hasAttributeValidationError: Object.values(attributeValidation).some((v) => v !== null),
    handleSubmit,
    toggleQuality,
    updateItemMode,
    updateField,
  }
}

type SubmitParams = {
  formData: CreateQualityProfileInput
  isEditing: boolean
  profile: QualityProfile | null | undefined
  createMutation: ReturnType<typeof useCreateQualityProfile>
  updateMutation: ReturnType<typeof useUpdateQualityProfile>
  onOpenChange: (open: boolean) => void
}

async function submitForm(params: SubmitParams) {
  const { formData, isEditing, profile, createMutation, updateMutation, onOpenChange } = params
  if (!formData.name.trim()) {
    toast.error('Name is required')
    return
  }
  if (formData.items.filter((i) => i.allowed).length === 0) {
    toast.error('At least one quality must be allowed')
    return
  }
  try {
    if (isEditing && profile) {
      await updateMutation.mutateAsync({ id: profile.id, data: formData })
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

type UpdateSettingsParams = {
  field: SettingsField
  value: string
  mode: AttributeMode
}

function buildUpdatedSettings(
  prev: CreateQualityProfileInput,
  params: UpdateSettingsParams,
): CreateQualityProfileInput {
  const { field, value, mode } = params
  if (mode === 'acceptable') {
    return { ...prev, [field]: { items: removeKey(prev[field].items, value) } }
  }
  const updated = { ...prev[field].items, [value]: mode }
  if (field === 'hdrSettings') {
    applyHdrMutualExclusion(updated, value, mode)
  }
  return { ...prev, [field]: { items: updated } }
}

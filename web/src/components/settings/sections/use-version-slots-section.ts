import { useEffect, useRef, useState } from 'react'

import { toast } from 'sonner'

import {
  useDeveloperMode,
  useImportSettings,
  useMultiVersionSettings,
  useQualityProfiles,
  useRootFoldersByType,
  useSetSlotEnabled,
  useSetSlotProfile,
  useSlots,
  useUpdateMultiVersionSettings,
  useUpdateSlot,
  useValidateNaming,
  useValidateSlotConfiguration,
} from '@/hooks'
import type { RootFolder, Slot, SlotConflict, SlotNamingValidation, UpdateSlotInput } from '@/types'

export type ValidationResult = {
  valid: boolean
  errors?: string[]
  conflicts?: SlotConflict[]
} | null

type ConfigReadyInput = {
  slots: Slot[] | undefined
  profiles: { id: number }[] | undefined
  movieRootFolders: RootFolder[] | undefined
  tvRootFolders: RootFolder[] | undefined
}

type IdSets = {
  profileIds: Set<number>
  movieIds: Set<number>
  tvIds: Set<number>
}

function isSlotReady(slot: Slot, ids: IdSets): boolean {
  const isRequired = slot.slotNumber <= 2 || slot.enabled
  if (!isRequired) {
    return true
  }
  if (!slot.qualityProfileId || !ids.profileIds.has(slot.qualityProfileId)) {
    return false
  }
  if (slot.movieRootFolderId !== null && !ids.movieIds.has(slot.movieRootFolderId)) {
    return false
  }
  if (slot.tvRootFolderId !== null && !ids.tvIds.has(slot.tvRootFolderId)) {
    return false
  }
  return true
}

function isConfigurationReady(input: ConfigReadyInput): boolean {
  if (!input.slots || !input.profiles || !input.movieRootFolders || !input.tvRootFolders) {
    return false
  }
  const ids: IdSets = {
    profileIds: new Set(input.profiles.map((p) => p.id)),
    movieIds: new Set(input.movieRootFolders.map((f) => f.id)),
    tvIds: new Set(input.tvRootFolders.map((f) => f.id)),
  }
  return input.slots.every((slot) => isSlotReady(slot, ids))
}

function useSlotQueries() {
  const slotsQuery = useSlots()
  const settingsQuery = useMultiVersionSettings()
  const profilesQuery = useQualityProfiles()
  const movieFoldersQuery = useRootFoldersByType('movie')
  const tvFoldersQuery = useRootFoldersByType('tv')
  const importSettingsQuery = useImportSettings()
  const developerMode = useDeveloperMode()

  return {
    slots: slotsQuery.data,
    slotsLoading: slotsQuery.isLoading,
    slotsError: slotsQuery.isError,
    refetchSlots: slotsQuery.refetch,
    settings: settingsQuery.data,
    settingsLoading: settingsQuery.isLoading,
    settingsError: settingsQuery.isError,
    refetchSettings: settingsQuery.refetch,
    profiles: profilesQuery.data,
    movieRootFolders: movieFoldersQuery.data,
    tvRootFolders: tvFoldersQuery.data,
    refetchImportSettings: importSettingsQuery.refetch,
    developerMode,
  }
}

function useSlotMutations() {
  return {
    updateSettings: useUpdateMultiVersionSettings(),
    updateSlot: useUpdateSlot(),
    setEnabled: useSetSlotEnabled(),
    setProfile: useSetSlotProfile(),
    validate: useValidateSlotConfiguration(),
    validateNaming: useValidateNaming(),
  }
}

type EnableMutate = (params: { id: number; data: { enabled: boolean } }) => void

function useAutoEnableSlots(slots: Slot[] | undefined, mutate: EnableMutate) {
  const initiated = useRef(false)
  useEffect(() => {
    if (slots && !initiated.current) {
      const toEnable = slots.filter((s) => s.slotNumber <= 2 && !s.enabled)
      if (toEnable.length > 0) {
        initiated.current = true
        for (const slot of toEnable) {
          mutate({ id: slot.id, data: { enabled: true } })
        }
      }
    }
  }, [slots, mutate])
}

function useSettingsHandlers(mutations: ReturnType<typeof useSlotMutations>) {
  const handleToggleMultiVersion = async (enabled: boolean) => {
    try {
      await mutations.updateSettings.mutateAsync({ enabled })
      toast.success(enabled ? 'Multi-version enabled' : 'Multi-version disabled')
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update settings'
      toast.error(message)
    }
  }

  const handleSlotProfileChange = async (slot: Slot, profileId: string) => {
    const id = profileId === 'none' ? null : Number.parseInt(profileId, 10)
    await mutations.setProfile.mutateAsync({ id: slot.id, data: { qualityProfileId: id } })
  }

  return { handleToggleMultiVersion, handleSlotProfileChange }
}

function useSlotUpdateHandlers(mutations: ReturnType<typeof useSlotMutations>) {
  const handleSlotEnabledChange = async (slot: Slot, enabled: boolean) => {
    if (enabled) {
      await mutations.setEnabled.mutateAsync({ id: slot.id, data: { enabled } })
      return
    }
    const input: UpdateSlotInput = {
      name: slot.name,
      enabled: false,
      qualityProfileId: null,
      displayOrder: slot.displayOrder,
      movieRootFolderId: null,
      tvRootFolderId: null,
    }
    await mutations.updateSlot.mutateAsync({ id: slot.id, data: input })
  }

  const handleSlotNameChange = async (slot: Slot, name: string) => {
    if (!name.trim()) {
      return
    }
    const input: UpdateSlotInput = {
      name: name.trim(),
      enabled: slot.enabled,
      qualityProfileId: slot.qualityProfileId,
      displayOrder: slot.displayOrder,
    }
    await mutations.updateSlot.mutateAsync({ id: slot.id, data: input })
  }

  const handleSlotRootFolderChange = async (
    slot: Slot,
    mediaType: 'movie' | 'tv',
    rootFolderId: string,
  ) => {
    const id = rootFolderId === 'none' ? null : Number.parseInt(rootFolderId, 10)
    const input: UpdateSlotInput = {
      name: slot.name,
      enabled: slot.enabled,
      qualityProfileId: slot.qualityProfileId,
      displayOrder: slot.displayOrder,
      movieRootFolderId: mediaType === 'movie' ? id : slot.movieRootFolderId,
      tvRootFolderId: mediaType === 'tv' ? id : slot.tvRootFolderId,
    }
    await mutations.updateSlot.mutateAsync({ id: slot.id, data: input })
  }

  return { handleSlotEnabledChange, handleSlotNameChange, handleSlotRootFolderChange }
}

function useValidationHandlers(
  mutations: ReturnType<typeof useSlotMutations>,
  refetchImportSettings: ReturnType<typeof useSlotQueries>['refetchImportSettings'],
) {
  const [validationResult, setValidationResult] = useState<ValidationResult>(null)
  const [namingValidation, setNamingValidation] = useState<SlotNamingValidation | null>(null)

  const handleValidate = async () => {
    try {
      const result = await mutations.validate.mutateAsync()
      setValidationResult(result)
      if (result.valid) {
        toast.success('Slot configuration is valid')
      } else {
        toast.error('Slot configuration has errors')
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Validation failed'
      toast.error(message)
    }
  }

  const handleValidateNaming = async () => {
    try {
      const { data: latestSettings } = await refetchImportSettings()
      const result = await mutations.validateNaming.mutateAsync({
        movieFileFormat:
          latestSettings?.movieFileFormat ?? '{Movie Title} ({Year}) - {Quality Title}',
        episodeFileFormat:
          latestSettings?.standardEpisodeFormat ??
          '{Series Title} - S{season:00}E{episode:00} - {Quality Title}',
      })
      setNamingValidation(result)
      if (result.canProceed) {
        toast.success('Filename formats are valid')
      } else {
        toast.warning('Filename formats may cause conflicts')
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Naming validation failed'
      toast.error(message)
    }
  }

  return { validationResult, namingValidation, handleValidate, handleValidateNaming }
}

function getQueryDefaults(queries: ReturnType<typeof useSlotQueries>) {
  return {
    profiles: queries.profiles ?? [],
    movieRootFolders: queries.movieRootFolders ?? [],
    tvRootFolders: queries.tvRootFolders ?? [],
    isLoading: queries.slotsLoading || queries.settingsLoading,
    isError: queries.slotsError || queries.settingsError,
    multiVersionEnabled: queries.settings?.enabled ?? false,
    configurationReady: isConfigurationReady(queries),
    enabledSlotCount: queries.slots?.filter((s) => s.enabled).length ?? 0,
  }
}

function getMutationState(mutations: ReturnType<typeof useSlotMutations>) {
  return {
    isSlotUpdating:
      mutations.setEnabled.isPending ||
      mutations.updateSlot.isPending ||
      mutations.setProfile.isPending,
    isTogglePending: mutations.updateSettings.isPending,
    isValidatePending: mutations.validate.isPending,
    isValidateNamingPending: mutations.validateNaming.isPending,
  }
}

function useDialogState() {
  const [infoCardDismissed, setInfoCardDismissed] = useState(false)
  const [resolveConfigOpen, setResolveConfigOpen] = useState(false)
  const [resolveNamingOpen, setResolveNamingOpen] = useState(false)
  const [dryRunOpen, setDryRunOpen] = useState(false)
  const [migrationError, setMigrationError] = useState<string | null>(null)

  return {
    infoCardDismissed,
    resolveConfigOpen,
    resolveNamingOpen,
    dryRunOpen,
    migrationError,
    setInfoCardDismissed,
    setResolveConfigOpen,
    setResolveNamingOpen,
    setDryRunOpen,
    setMigrationError,
  }
}

export function useVersionSlotsSection() {
  const queries = useSlotQueries()
  const mutations = useSlotMutations()

  useAutoEnableSlots(queries.slots, mutations.setEnabled.mutate)

  return {
    slots: queries.slots,
    settings: queries.settings,
    developerMode: queries.developerMode,
    ...getQueryDefaults(queries),
    ...getMutationState(mutations),
    ...useValidationHandlers(mutations, queries.refetchImportSettings),
    ...useDialogState(),
    handleRetry: () => {
      void queries.refetchSlots()
      void queries.refetchSettings()
    },
    ...useSettingsHandlers(mutations),
    ...useSlotUpdateHandlers(mutations),
    handleMigrationComplete: () => {
      void queries.refetchSettings()
      void queries.refetchSlots()
    },
  }
}

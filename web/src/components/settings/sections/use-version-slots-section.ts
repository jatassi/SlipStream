import { useEffect, useRef, useState } from 'react'

import { toast } from 'sonner'

import {
  useDeveloperMode,
  useModuleNamingSettings,
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
  rootFoldersByModule: Record<string, RootFolder[]>
}

function isSlotReady(slot: Slot, profileIds: Set<number>, rootFolderIdsByModule: Map<string, Set<number>>): boolean {
  const isRequired = slot.slotNumber <= 2 || slot.enabled
  if (!isRequired) {
    return true
  }
  if (!slot.qualityProfileId || !profileIds.has(slot.qualityProfileId)) {
    return false
  }
  for (const [moduleType, folderId] of Object.entries(slot.rootFolders)) {
    const moduleIds = rootFolderIdsByModule.get(moduleType)
    if (folderId !== null && !moduleIds?.has(folderId)) {
      return false
    }
  }
  return true
}

function isConfigurationReady(input: ConfigReadyInput): boolean {
  if (!input.slots || !input.profiles) {
    return false
  }
  const profileIds = new Set(input.profiles.map((p) => p.id))
  const rootFolderIdsByModule = new Map<string, Set<number>>()
  for (const [moduleType, folders] of Object.entries(input.rootFoldersByModule)) {
    rootFolderIdsByModule.set(moduleType, new Set(folders.map((f) => f.id)))
  }
  return input.slots.every((slot) => isSlotReady(slot, profileIds, rootFolderIdsByModule))
}

function useSlotQueries() {
  const slotsQuery = useSlots()
  const settingsQuery = useMultiVersionSettings()
  const profilesQuery = useQualityProfiles()
  const movieFoldersQuery = useRootFoldersByType('movie')
  const tvFoldersQuery = useRootFoldersByType('tv')
  const movieNamingQuery = useModuleNamingSettings('movie')
  const tvNamingQuery = useModuleNamingSettings('tv')
  const developerMode = useDeveloperMode()

  const rootFoldersByModule: Record<string, RootFolder[]> = {
    movie: movieFoldersQuery.data ?? [],
    tv: tvFoldersQuery.data ?? [],
  }

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
    rootFoldersByModule,
    refetchMovieNaming: movieNamingQuery.refetch,
    refetchTvNaming: tvNamingQuery.refetch,
    movieNaming: movieNamingQuery.data,
    tvNaming: tvNamingQuery.data,
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
      rootFolders: {},
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
    moduleType: string,
    rootFolderId: string,
  ) => {
    const id = rootFolderId === 'none' ? null : Number.parseInt(rootFolderId, 10)
    const input: UpdateSlotInput = {
      name: slot.name,
      enabled: slot.enabled,
      qualityProfileId: slot.qualityProfileId,
      displayOrder: slot.displayOrder,
      rootFolders: { ...slot.rootFolders, [moduleType]: id },
    }
    await mutations.updateSlot.mutateAsync({ id: slot.id, data: input })
  }

  return { handleSlotEnabledChange, handleSlotNameChange, handleSlotRootFolderChange }
}

function useValidationHandlers(
  mutations: ReturnType<typeof useSlotMutations>,
  refetchNaming: { movie: ReturnType<typeof useSlotQueries>['refetchMovieNaming']; tv: ReturnType<typeof useSlotQueries>['refetchTvNaming'] },
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
      const [{ data: movieNaming }, { data: tvNaming }] = await Promise.all([
        refetchNaming.movie(),
        refetchNaming.tv(),
      ])
      const result = await mutations.validateNaming.mutateAsync({
        movieFileFormat:
          movieNaming?.patterns['movie-file'] ?? '{Movie Title} ({Year}) - {Quality Title}',
        episodeFileFormat:
          tvNaming?.patterns['episode-file.standard'] ??
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
    rootFoldersByModule: queries.rootFoldersByModule,
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
    ...useValidationHandlers(mutations, { movie: queries.refetchMovieNaming, tv: queries.refetchTvNaming }),
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

import { useEffect, useMemo, useRef, useState } from 'react'

import {
  AlertTriangle,
  Check,
  FileText,
  FlaskConical,
  Info,
  Layers,
  Pencil,
  Settings,
  Wand2,
  X,
  XCircle,
} from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/ErrorState'
import { LoadingState } from '@/components/data/LoadingState'
import {
  DryRunModal,
  ResolveConfigModal,
  ResolveNamingModal,
  SlotDebugPanel,
} from '@/components/slots'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
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

export function VersionSlotsSection() {
  const {
    data: slots,
    isLoading: slotsLoading,
    isError: slotsError,
    refetch: refetchSlots,
  } = useSlots()
  const {
    data: settings,
    isLoading: settingsLoading,
    isError: settingsError,
    refetch: refetchSettings,
  } = useMultiVersionSettings()
  const { data: profiles } = useQualityProfiles()
  const { data: movieRootFolders } = useRootFoldersByType('movie')
  const { data: tvRootFolders } = useRootFoldersByType('tv')
  const { refetch: refetchImportSettings } = useImportSettings()
  const developerMode = useDeveloperMode()

  const updateSettingsMutation = useUpdateMultiVersionSettings()
  const updateSlotMutation = useUpdateSlot()
  const setEnabledMutation = useSetSlotEnabled()
  const setProfileMutation = useSetSlotProfile()
  const validateMutation = useValidateSlotConfiguration()
  const validateNamingMutation = useValidateNaming()

  // Auto-enable slots 1 and 2 if they're not already enabled
  const autoEnableInitiated = useRef(false)
  const setEnabledMutate = setEnabledMutation.mutate
  useEffect(() => {
    if (slots && !autoEnableInitiated.current) {
      const slotsToEnable = slots.filter((s) => s.slotNumber <= 2 && !s.enabled)
      if (slotsToEnable.length > 0) {
        autoEnableInitiated.current = true
        for (const slot of slotsToEnable) {
          setEnabledMutate({ id: slot.id, data: { enabled: true } })
        }
      }
    }
  }, [slots, setEnabledMutate])

  const [validationResult, setValidationResult] = useState<{
    valid: boolean
    errors?: string[]
    conflicts?: SlotConflict[]
  } | null>(null)
  const [namingValidation, setNamingValidation] = useState<SlotNamingValidation | null>(null)
  const [infoCardDismissed, setInfoCardDismissed] = useState(false)
  const [resolveConfigOpen, setResolveConfigOpen] = useState(false)
  const [resolveNamingOpen, setResolveNamingOpen] = useState(false)
  const [dryRunOpen, setDryRunOpen] = useState(false)
  const [migrationError, setMigrationError] = useState<string | null>(null)

  const multiVersionEnabled = settings?.enabled ?? false

  const configurationReady = useMemo(() => {
    if (!slots || !profiles || !movieRootFolders || !tvRootFolders) {
      return false
    }

    const profileIds = new Set(profiles.map((p) => p.id))
    const movieRootFolderIds = new Set(movieRootFolders.map((f) => f.id))
    const tvRootFolderIds = new Set(tvRootFolders.map((f) => f.id))

    for (const slot of slots) {
      const isRequired = slot.slotNumber <= 2 || slot.enabled

      if (isRequired) {
        if (!slot.qualityProfileId || !profileIds.has(slot.qualityProfileId)) {
          return false
        }
        if (slot.movieRootFolderId !== null && !movieRootFolderIds.has(slot.movieRootFolderId)) {
          return false
        }
        if (slot.tvRootFolderId !== null && !tvRootFolderIds.has(slot.tvRootFolderId)) {
          return false
        }
      }
    }

    return true
  }, [slots, profiles, movieRootFolders, tvRootFolders])

  const handleToggleMultiVersion = async (enabled: boolean) => {
    try {
      await updateSettingsMutation.mutateAsync({ enabled })
      toast.success(enabled ? 'Multi-version enabled' : 'Multi-version disabled')
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update settings'
      toast.error(message)
    }
  }

  const handleSlotEnabledChange = async (slot: Slot, enabled: boolean) => {
    if (enabled) {
      await setEnabledMutation.mutateAsync({ id: slot.id, data: { enabled } })
    } else {
      const input: UpdateSlotInput = {
        name: slot.name,
        enabled: false,
        qualityProfileId: null,
        displayOrder: slot.displayOrder,
        movieRootFolderId: null,
        tvRootFolderId: null,
      }
      await updateSlotMutation.mutateAsync({ id: slot.id, data: input })
    }
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
    await updateSlotMutation.mutateAsync({ id: slot.id, data: input })
  }

  const handleSlotProfileChange = async (slot: Slot, profileId: string) => {
    const id = profileId === 'none' ? null : Number.parseInt(profileId, 10)
    await setProfileMutation.mutateAsync({ id: slot.id, data: { qualityProfileId: id } })
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
    await updateSlotMutation.mutateAsync({ id: slot.id, data: input })
  }

  const handleValidate = async () => {
    try {
      const result = await validateMutation.mutateAsync()
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

      const result = await validateNamingMutation.mutateAsync({
        movieFileFormat:
          latestSettings?.movieFileFormat || '{Movie Title} ({Year}) - {Quality Title}',
        episodeFileFormat:
          latestSettings?.standardEpisodeFormat ||
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

  const isLoading = slotsLoading || settingsLoading
  const isError = slotsError || settingsError

  if (isLoading) {
    return <LoadingState variant="list" count={3} />
  }

  if (isError) {
    return (
      <ErrorState
        onRetry={() => {
          refetchSlots()
          refetchSettings()
        }}
      />
    )
  }

  const enabledSlotCount = slots?.filter((s) => s.enabled).length ?? 0

  return (
    <div className="space-y-6">
      {!infoCardDismissed && !multiVersionEnabled && (
        <Alert className="border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950/50">
          <Info className="size-4 text-blue-600 dark:text-blue-400" />
          <AlertTitle className="flex items-center justify-between">
            <span>Multi-Version Feature</span>
            <Button
              variant="ghost"
              size="icon"
              className="-mt-1 -mr-2 size-6"
              onClick={() => setInfoCardDismissed(true)}
            >
              <X className="size-4" />
            </Button>
          </AlertTitle>
          <AlertDescription className="text-blue-800 dark:text-blue-200">
            <ul className="mt-1 list-inside list-disc space-y-1">
              <li>Keep multiple quality versions of the same media (e.g., 4K HDR and 1080p SDR)</li>
              <li>Assign a different quality profile to each slot</li>
              <li>Files are downloaded and organized separately for each slot</li>
            </ul>
          </AlertDescription>
        </Alert>
      )}

      {/* Master Toggle */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <CardTitle className="flex items-center gap-2">
                <Layers className="size-5" />
                Multi-Version Mode
              </CardTitle>
              <CardDescription>
                {settings?.enabled
                  ? `Active with ${enabledSlotCount} slot${enabledSlotCount === 1 ? '' : 's'} enabled`
                  : 'Use the dry run below to preview and enable multi-version mode'}
              </CardDescription>
            </div>
            {multiVersionEnabled ? (
              <Switch
                id="multi-version-toggle"
                checked
                onCheckedChange={handleToggleMultiVersion}
                disabled={updateSettingsMutation.isPending}
                className="origin-right scale-150"
              />
            ) : (
              <Tooltip>
                <TooltipTrigger>
                  <Switch
                    id="multi-version-toggle"
                    checked={false}
                    disabled
                    className="origin-right scale-150"
                  />
                </TooltipTrigger>
                <TooltipContent side="left">
                  <p>Perform a dry run first</p>
                </TooltipContent>
              </Tooltip>
            )}
          </div>
        </CardHeader>
        {migrationError || !settings?.enabled ? (
          <CardContent className="space-y-3">
            {migrationError ? (
              <Alert className="border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/50">
                <XCircle className="size-4 text-red-600 dark:text-red-400" />
                <AlertTitle className="flex items-center justify-between">
                  <span className="text-red-800 dark:text-red-200">
                    Failed to Enable Multi-Version Mode
                  </span>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="-mt-1 -mr-2 size-6"
                    onClick={() => setMigrationError(null)}
                  >
                    <X className="size-4" />
                  </Button>
                </AlertTitle>
                <AlertDescription className="text-red-700 dark:text-red-300">
                  <p className="mt-1">{migrationError}</p>
                  <p className="mt-2 text-sm">
                    Try running the dry run again to review your file assignments, or check that
                    your slot configuration is valid.
                  </p>
                </AlertDescription>
              </Alert>
            ) : null}

            {!settings?.enabled && (
              <Alert className="border-purple-300 bg-purple-50 dark:border-purple-700 dark:bg-purple-950/50">
                <FlaskConical className="size-4 text-purple-600 dark:text-purple-400" />
                <div className="flex flex-1 items-center justify-between">
                  <div>
                    <AlertTitle>Dry Run</AlertTitle>
                    <AlertDescription>
                      See how your existing files will be organized
                    </AlertDescription>
                  </div>
                  <Button onClick={() => setDryRunOpen(true)} disabled={!configurationReady}>
                    Begin
                  </Button>
                </div>
              </Alert>
            )}

            {!settings?.enabled && (
              <Alert
                className={
                  validationResult === null
                    ? ''
                    : validationResult.valid
                      ? 'border-green-500 dark:border-green-600'
                      : 'border-orange-400 dark:border-orange-500'
                }
              >
                {validationResult === null ? (
                  <Settings className="size-4" />
                ) : validationResult.valid ? (
                  <Check className="size-4 text-green-600 dark:text-green-400" />
                ) : (
                  <AlertTriangle className="size-4 text-orange-500 dark:text-orange-400" />
                )}
                <div className="flex flex-1 items-center justify-between">
                  <div className="flex-1">
                    <AlertTitle>
                      Quality Profiles{' '}
                      {validationResult === null
                        ? 'Validation'
                        : validationResult.valid
                          ? 'Valid'
                          : 'Not Valid'}
                    </AlertTitle>
                    <AlertDescription>
                      {validationResult === null ? (
                        'Check that assigned Quality Profiles are mutually exclusive'
                      ) : validationResult.valid ? (
                        <div className="mt-2">
                          <span>Slot profiles are mutually exclusive based on one or more of:</span>
                          <ul className="text-muted-foreground mt-1 ml-4 space-y-0.5 text-sm">
                            <li>• Different allowed quality tiers (e.g., 1080p vs 2160p)</li>
                            <li>
                              • Conflicting HDR requirements (e.g., HDR required vs SDR required)
                            </li>
                            <li>• Conflicting video codec requirements</li>
                            <li>• Conflicting audio codec or channel requirements</li>
                          </ul>
                        </div>
                      ) : (
                        <div className="mt-2">
                          {validationResult.errors?.some(
                            (e) => !e.startsWith('Profile conflict'),
                          ) ? (
                            <ul className="list-inside list-disc">
                              {validationResult.errors
                                .filter((e) => !e.startsWith('Profile conflict'))
                                .map((error, i) => (
                                  <li key={i}>{error}</li>
                                ))}
                            </ul>
                          ) : null}
                          {validationResult.conflicts && validationResult.conflicts.length > 0 ? (
                            <div className="mt-2 space-y-3">
                              {validationResult.conflicts.map((conflict, i) => (
                                <div key={i}>
                                  <p className="font-medium">
                                    Conflict between {conflict.slotAName} and {conflict.slotBName}:
                                  </p>
                                  <ul className="mt-1 ml-4 list-inside list-disc space-y-0.5">
                                    {conflict.issues.map((issue, j) => (
                                      <li key={j}>
                                        <span className="font-medium">{issue.attribute}:</span>{' '}
                                        {issue.message}
                                      </li>
                                    ))}
                                  </ul>
                                </div>
                              ))}
                            </div>
                          ) : null}
                        </div>
                      )}
                    </AlertDescription>
                  </div>
                  {validationResult !== null && !validationResult.valid ? (
                    <Button
                      onClick={() => setResolveConfigOpen(true)}
                      className="ml-4 shrink-0 bg-orange-500 text-white hover:bg-orange-600"
                    >
                      <Wand2 className="mr-2 size-4" />
                      Resolve...
                    </Button>
                  ) : (
                    <Button
                      variant="outline"
                      onClick={handleValidate}
                      disabled={!configurationReady || validateMutation.isPending}
                      className="ml-4 shrink-0"
                    >
                      {validateMutation.isPending ? 'Validating...' : 'Validate'}
                    </Button>
                  )}
                </div>
              </Alert>
            )}

            {!settings?.enabled && (
              <Alert
                className={
                  namingValidation === null
                    ? ''
                    : namingValidation.noEnabledSlots
                      ? 'border-orange-400 dark:border-orange-500'
                      : namingValidation.canProceed
                        ? 'border-green-500 dark:border-green-600'
                        : 'border-orange-400 dark:border-orange-500'
                }
              >
                {namingValidation === null ? (
                  <FileText className="size-4" />
                ) : namingValidation.noEnabledSlots ? (
                  <AlertTriangle className="size-4 text-orange-500 dark:text-orange-400" />
                ) : namingValidation.canProceed ? (
                  <Check className="size-4 text-green-600 dark:text-green-400" />
                ) : (
                  <AlertTriangle className="size-4 text-orange-500 dark:text-orange-400" />
                )}
                <div className="flex flex-1 items-center justify-between">
                  <div className="flex-1">
                    <AlertTitle>
                      File Naming{' '}
                      {namingValidation === null
                        ? 'Validation'
                        : namingValidation.noEnabledSlots
                          ? 'Blocked'
                          : namingValidation.canProceed
                            ? 'Valid'
                            : 'Not Valid'}
                    </AlertTitle>
                    <AlertDescription>
                      {namingValidation === null ? (
                        'Verify filename formats include required differentiator tokens'
                      ) : namingValidation.noEnabledSlots ? (
                        <p className="mt-2 text-orange-700 dark:text-orange-300">
                          Complete Quality Profile validation first to ensure slot profiles are
                          mutually exclusive.
                        </p>
                      ) : namingValidation.canProceed ? (
                        <p className="mt-2">
                          {(namingValidation.requiredAttributes?.length ?? 0) === 0
                            ? namingValidation.qualityTierExclusive
                              ? 'Profiles are distinguished by quality tier - no additional filename tokens required'
                              : 'Complete Quality Profile validation first'
                            : `File name formats include tokens for differentiating attributes: ${namingValidation.requiredAttributes.join(', ')}`}
                        </p>
                      ) : (
                        <div className="mt-2 space-y-3">
                          <p>
                            Slots have different requirements for:{' '}
                            <span className="font-medium">
                              {namingValidation.requiredAttributes?.join(', ') ?? ''}
                            </span>
                          </p>
                          {!namingValidation.movieFormatValid &&
                          namingValidation.movieValidation.missingTokens ? (
                            <div>
                              <p className="font-medium">Movie filename format missing tokens:</p>
                              <ul className="mt-1 ml-4 list-inside list-disc space-y-0.5">
                                {namingValidation.movieValidation.missingTokens.map((token, i) => (
                                  <li key={i}>
                                    <span className="font-medium">{token.attribute}:</span> Add{' '}
                                    <code className="bg-muted rounded px-1">
                                      {token.suggestedToken}
                                    </code>
                                  </li>
                                ))}
                              </ul>
                            </div>
                          ) : null}
                          {!namingValidation.episodeFormatValid &&
                          namingValidation.episodeValidation.missingTokens ? (
                            <div>
                              <p className="font-medium">Episode filename format missing tokens:</p>
                              <ul className="mt-1 ml-4 list-inside list-disc space-y-0.5">
                                {namingValidation.episodeValidation.missingTokens.map(
                                  (token, i) => (
                                    <li key={i}>
                                      <span className="font-medium">{token.attribute}:</span> Add{' '}
                                      <code className="bg-muted rounded px-1">
                                        {token.suggestedToken}
                                      </code>
                                    </li>
                                  ),
                                )}
                              </ul>
                            </div>
                          ) : null}
                        </div>
                      )}
                    </AlertDescription>
                  </div>
                  {namingValidation !== null && !namingValidation.canProceed ? (
                    <Button
                      onClick={() => setResolveNamingOpen(true)}
                      className="ml-4 shrink-0 bg-orange-500 text-white hover:bg-orange-600"
                    >
                      <Wand2 className="mr-2 size-4" />
                      Resolve...
                    </Button>
                  ) : (
                    <Button
                      variant="outline"
                      onClick={handleValidateNaming}
                      disabled={!configurationReady || validateNamingMutation.isPending}
                      className="ml-4 shrink-0"
                    >
                      {validateNamingMutation.isPending ? 'Validating...' : 'Validate'}
                    </Button>
                  )}
                </div>
              </Alert>
            )}
          </CardContent>
        ) : null}
      </Card>

      {/* Slot Configuration */}
      <div className="grid gap-4 md:grid-cols-3">
        {slots?.map((slot) => {
          const usedProfileIds = slots
            .filter((s) => s.id !== slot.id && s.qualityProfileId !== null)
            .map((s) => s.qualityProfileId!)
          return (
            <SlotCard
              key={slot.id}
              slot={slot}
              profiles={profiles ?? []}
              usedProfileIds={usedProfileIds}
              movieRootFolders={movieRootFolders ?? []}
              tvRootFolders={tvRootFolders ?? []}
              onEnabledChange={(enabled) => handleSlotEnabledChange(slot, enabled)}
              onNameChange={(name) => handleSlotNameChange(slot, name)}
              onProfileChange={(profileId) => handleSlotProfileChange(slot, profileId)}
              onRootFolderChange={(mediaType, rootFolderId) =>
                handleSlotRootFolderChange(slot, mediaType, rootFolderId)
              }
              isUpdating={
                setEnabledMutation.isPending ||
                updateSlotMutation.isPending ||
                setProfileMutation.isPending
              }
              showToggle={slot.slotNumber === 3}
            />
          )
        })}
      </div>

      {developerMode ? <SlotDebugPanel /> : null}

      <ResolveConfigModal
        open={resolveConfigOpen}
        onOpenChange={setResolveConfigOpen}
        conflicts={validationResult?.conflicts || []}
        onResolved={handleValidate}
      />

      <ResolveNamingModal
        open={resolveNamingOpen}
        onOpenChange={setResolveNamingOpen}
        missingMovieTokens={namingValidation?.movieValidation?.missingTokens}
        missingEpisodeTokens={namingValidation?.episodeValidation?.missingTokens}
        onResolved={handleValidateNaming}
      />

      <DryRunModal
        open={dryRunOpen}
        onOpenChange={setDryRunOpen}
        onMigrationComplete={() => {
          refetchSettings()
          refetchSlots()
        }}
        onMigrationFailed={(error) => setMigrationError(error)}
      />
    </div>
  )
}

type SlotCardProps = {
  slot: Slot
  profiles: { id: number; name: string }[]
  usedProfileIds: number[]
  movieRootFolders: RootFolder[]
  tvRootFolders: RootFolder[]
  onEnabledChange: (enabled: boolean) => void
  onNameChange: (name: string) => void
  onProfileChange: (profileId: string) => void
  onRootFolderChange: (mediaType: 'movie' | 'tv', rootFolderId: string) => void
  isUpdating: boolean
  showToggle?: boolean
}

function SlotCard({
  slot,
  profiles,
  usedProfileIds,
  movieRootFolders,
  tvRootFolders,
  onEnabledChange,
  onNameChange,
  onProfileChange,
  onRootFolderChange,
  isUpdating,
  showToggle = false,
}: SlotCardProps) {
  const availableProfiles = profiles.filter((p) => !usedProfileIds.includes(p.id))
  const [editingName, setEditingName] = useState(false)
  const [tempName, setTempName] = useState(slot.name)

  const isActive = slot.enabled

  const handleNameSubmit = () => {
    if (tempName.trim() && tempName !== slot.name) {
      onNameChange(tempName.trim())
    }
    setEditingName(false)
  }

  return (
    <Card className={isActive ? 'ring-primary/50 ring-2' : ''}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between gap-2">
          <div className="flex shrink-0 items-center gap-2">
            <div className="bg-primary/10 text-primary flex h-8 w-8 items-center justify-center rounded-full font-semibold">
              {slot.slotNumber}
            </div>
          </div>
          <div className="min-w-0 flex-1">
            <div
              className={`group relative flex items-center rounded-md border transition-colors ${
                editingName
                  ? 'border-primary bg-background'
                  : 'hover:border-muted-foreground/25 hover:bg-muted/50 border-transparent'
              }`}
            >
              <Input
                value={tempName}
                onChange={(e) => setTempName(e.target.value)}
                onFocus={() => setEditingName(true)}
                onBlur={handleNameSubmit}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    handleNameSubmit()
                    e.currentTarget.blur()
                  }
                  if (e.key === 'Escape') {
                    setTempName(slot.name)
                    setEditingName(false)
                    e.currentTarget.blur()
                  }
                }}
                className="h-8 border-0 bg-transparent pr-8 text-base font-semibold tracking-tight focus-visible:ring-0 focus-visible:ring-offset-0"
              />
              <Pencil
                className={`text-muted-foreground absolute right-2 size-3.5 transition-opacity ${
                  editingName ? 'opacity-0' : 'opacity-0 group-hover:opacity-100'
                }`}
              />
            </div>
          </div>
          {showToggle ? (
            slot.qualityProfileId === null ? (
              <Tooltip>
                <TooltipTrigger>
                  <Switch checked={slot.enabled} disabled className="shrink-0" />
                </TooltipTrigger>
                <TooltipContent side="left">
                  <p>Select a quality profile first</p>
                </TooltipContent>
              </Tooltip>
            ) : (
              <Switch
                checked={slot.enabled}
                onCheckedChange={onEnabledChange}
                disabled={isUpdating}
                className="shrink-0"
              />
            )
          ) : null}
        </div>
        <CardDescription>
          {isActive ? 'Active' : 'Disabled'}
          {slot.fileCount !== undefined && slot.fileCount > 0 && ` • ${slot.fileCount} files`}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor={`slot-${slot.id}-profile`}>Quality Profile</Label>
            <Select
              value={slot.qualityProfileId?.toString() ?? 'none'}
              onValueChange={(v) => v && onProfileChange(v)}
              disabled={isUpdating}
            >
              <SelectTrigger id={`slot-${slot.id}-profile`}>
                {slot.qualityProfile?.name ?? 'Select profile...'}
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">None</SelectItem>
                {availableProfiles.map((profile) => (
                  <SelectItem key={profile.id} value={profile.id.toString()}>
                    {profile.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor={`slot-${slot.id}-movie-root`}>Movie Root Folder</Label>
            <Select
              value={slot.movieRootFolderId?.toString() ?? 'none'}
              onValueChange={(v) => v && onRootFolderChange('movie', v)}
              disabled={isUpdating}
            >
              <SelectTrigger id={`slot-${slot.id}-movie-root`}>
                {movieRootFolders.find((f) => f.id === slot.movieRootFolderId)?.name ??
                  'Use media default'}
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">Use media default</SelectItem>
                {movieRootFolders.map((folder) => (
                  <SelectItem key={folder.id} value={folder.id.toString()}>
                    {folder.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor={`slot-${slot.id}-tv-root`}>TV Root Folder</Label>
            <Select
              value={slot.tvRootFolderId?.toString() ?? 'none'}
              onValueChange={(v) => v && onRootFolderChange('tv', v)}
              disabled={isUpdating}
            >
              <SelectTrigger id={`slot-${slot.id}-tv-root`}>
                {tvRootFolders.find((f) => f.id === slot.tvRootFolderId)?.name ??
                  'Use media default'}
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">Use media default</SelectItem>
                {tvRootFolders.map((folder) => (
                  <SelectItem key={folder.id} value={folder.id.toString()}>
                    {folder.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

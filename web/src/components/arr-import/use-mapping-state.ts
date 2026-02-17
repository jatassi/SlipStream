import { useState } from 'react'

import type { ImportMappings, SourceQualityProfile, SourceRootFolder } from '@/types/arr-import'

function autoMatchRootFolder(
  sourcePath: string,
  targetFolders: { id: number; path: string }[],
): number | undefined {
  const normalizedSource = sourcePath.toLowerCase()
  const exactMatch = targetFolders.find((f) => f.path.toLowerCase() === normalizedSource)
  if (exactMatch) {
    return exactMatch.id
  }

  const partialMatch = targetFolders.find(
    (f) =>
      f.path.toLowerCase().includes(normalizedSource) ||
      normalizedSource.includes(f.path.toLowerCase()),
  )
  return partialMatch?.id
}

function autoMatchQualityProfile(
  sourceName: string,
  targetProfiles: { id: number; name: string }[],
): number | undefined {
  const normalizedSource = sourceName.toLowerCase()
  const exactMatch = targetProfiles.find((p) => p.name.toLowerCase() === normalizedSource)
  if (exactMatch) {
    return exactMatch.id
  }

  const partialMatch = targetProfiles.find(
    (p) =>
      p.name.toLowerCase().includes(normalizedSource) ||
      normalizedSource.includes(p.name.toLowerCase()),
  )
  return partialMatch?.id
}

function buildRootFolderAutoMapping(
  sourceRootFolders: SourceRootFolder[],
  targetRootFolders: { id: number; path: string }[],
): Record<string, number> {
  const autoMapping: Record<string, number> = {}
  for (const sourceFolder of sourceRootFolders) {
    const match = autoMatchRootFolder(sourceFolder.path, targetRootFolders)
    if (match !== undefined) {
      autoMapping[sourceFolder.path] = match
    }
  }
  return autoMapping
}

function buildQualityProfileAutoMapping(
  sourceQualityProfiles: SourceQualityProfile[],
  targetQualityProfiles: { id: number; name: string }[],
): Record<number, number> {
  const autoMapping: Record<number, number> = {}
  for (const sourceProfile of sourceQualityProfiles) {
    const match = autoMatchQualityProfile(sourceProfile.name, targetQualityProfiles)
    if (match !== undefined) {
      autoMapping[sourceProfile.id] = match
    }
  }
  return autoMapping
}

function buildInitialProfileEnabled(profiles: SourceQualityProfile[]): Record<number, boolean> {
  const enabled: Record<number, boolean> = {}
  for (const profile of profiles) {
    enabled[profile.id] = profile.inUse
  }
  return enabled
}

type UseMappingStateOptions = {
  sourceRootFolders: SourceRootFolder[]
  sourceQualityProfiles: SourceQualityProfile[]
  targetRootFolders: { id: number; path: string }[] | undefined
  targetQualityProfiles: { id: number; name: string }[] | undefined
}

function buildEnabledProfileMapping(
  profiles: SourceQualityProfile[],
  profileEnabled: Record<number, boolean>,
  qualityProfileMapping: Record<number, number>,
): Record<number, number> {
  const filtered: Record<number, number> = {}
  for (const profile of profiles) {
    if (profileEnabled[profile.id] && profile.id in qualityProfileMapping) {
      filtered[profile.id] = qualityProfileMapping[profile.id]
    }
  }
  return filtered
}

export function useMappingState({
  sourceRootFolders,
  sourceQualityProfiles,
  targetRootFolders,
  targetQualityProfiles,
}: UseMappingStateOptions) {
  const [rootFolderMapping, setRootFolderMapping] = useState<Record<string, number>>({})
  const [qualityProfileMapping, setQualityProfileMapping] = useState<Record<number, number>>({})
  const [profileEnabled, setProfileEnabled] = useState<Record<number, boolean>>(() =>
    buildInitialProfileEnabled(sourceQualityProfiles),
  )

  const [prevTargetRootFolders, setPrevTargetRootFolders] = useState(targetRootFolders)
  const [prevTargetQualityProfiles, setPrevTargetQualityProfiles] = useState(targetQualityProfiles)

  if (targetRootFolders !== prevTargetRootFolders) {
    setPrevTargetRootFolders(targetRootFolders)
    if (targetRootFolders) {
      setRootFolderMapping(buildRootFolderAutoMapping(sourceRootFolders, targetRootFolders))
    }
  }

  if (targetQualityProfiles !== prevTargetQualityProfiles) {
    setPrevTargetQualityProfiles(targetQualityProfiles)
    if (targetQualityProfiles) {
      setQualityProfileMapping(
        buildQualityProfileAutoMapping(sourceQualityProfiles, targetQualityProfiles),
      )
    }
  }

  const allRootFoldersMapped = sourceRootFolders.every((folder) => folder.path in rootFolderMapping)
  const allProfilesMapped = sourceQualityProfiles.every(
    (profile) => !profileEnabled[profile.id] || profile.id in qualityProfileMapping,
  )

  const handleNext = (onMappingsComplete: (mappings: ImportMappings) => void) => {
    if (!allRootFoldersMapped || !allProfilesMapped) {
      return
    }
    const filtered = buildEnabledProfileMapping(sourceQualityProfiles, profileEnabled, qualityProfileMapping)
    onMappingsComplete({ rootFolderMapping, qualityProfileMapping: filtered })
  }

  return { rootFolderMapping, setRootFolderMapping, qualityProfileMapping, setQualityProfileMapping, profileEnabled, setProfileEnabled, allRootFoldersMapped, allProfilesMapped, handleNext }
}

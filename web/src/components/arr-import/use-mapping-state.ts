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

type UseMappingStateOptions = {
  sourceRootFolders: SourceRootFolder[]
  sourceQualityProfiles: SourceQualityProfile[]
  targetRootFolders: { id: number; path: string }[] | undefined
  targetQualityProfiles: { id: number; name: string }[] | undefined
}

export function useMappingState({
  sourceRootFolders,
  sourceQualityProfiles,
  targetRootFolders,
  targetQualityProfiles,
}: UseMappingStateOptions) {
  const [rootFolderMapping, setRootFolderMapping] = useState<Record<string, number>>({})
  const [qualityProfileMapping, setQualityProfileMapping] = useState<Record<number, number>>({})

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
      setQualityProfileMapping(buildQualityProfileAutoMapping(sourceQualityProfiles, targetQualityProfiles))
    }
  }

  const allRootFoldersMapped = sourceRootFolders.every((folder) => folder.path in rootFolderMapping)
  const allProfilesMapped = sourceQualityProfiles.every((profile) => profile.id in qualityProfileMapping)

  const handleNext = (onMappingsComplete: (mappings: ImportMappings) => void) => {
    if (!allRootFoldersMapped || !allProfilesMapped) {
      return
    }
    onMappingsComplete({ rootFolderMapping, qualityProfileMapping })
  }

  return {
    rootFolderMapping,
    setRootFolderMapping,
    qualityProfileMapping,
    setQualityProfileMapping,
    allRootFoldersMapped,
    allProfilesMapped,
    handleNext,
  }
}

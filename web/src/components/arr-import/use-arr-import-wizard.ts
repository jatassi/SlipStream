import { useState } from 'react'

import {
  useDisconnect,
  useExecuteImport,
  usePreview,
  useSourceQualityProfiles,
  useSourceRootFolders,
  type WizardStep,
} from '@/hooks/use-arr-import'
import type {
  ImportMappings,
  ImportPreview,
  SourceQualityProfile,
  SourceRootFolder,
  SourceType,
} from '@/types/arr-import'

type WizardState = {
  currentStep: WizardStep
  sourceType: SourceType
  mappings: ImportMappings | null
  preview: ImportPreview | null
  sourceRootFolders: SourceRootFolder[]
  sourceQualityProfiles: SourceQualityProfile[]
  isLoadingSourceData: boolean
  completedSteps: Set<WizardStep>
}

const initialState: WizardState = {
  currentStep: 'connect',
  sourceType: 'radarr',
  mappings: null,
  preview: null,
  sourceRootFolders: [],
  sourceQualityProfiles: [],
  isLoadingSourceData: false,
  completedSteps: new Set(),
}

type SetState = React.Dispatch<React.SetStateAction<WizardState>>

function createHandleConnected(
  setState: SetState,
  refetchRootFolders: () => Promise<{ data?: SourceRootFolder[] }>,
  refetchQualityProfiles: () => Promise<{ data?: SourceQualityProfile[] }>,
) {
  return async () => {
    setState((s) => ({ ...s, isLoadingSourceData: true }))
    try {
      const [rfResult, qpResult] = await Promise.all([refetchRootFolders(), refetchQualityProfiles()])

      const rootFolders = rfResult.data
      const qualityProfiles = qpResult.data
      if (rootFolders && qualityProfiles) {
        setState((s) => ({
          ...s,
          sourceRootFolders: rootFolders,
          sourceQualityProfiles: qualityProfiles,
          currentStep: 'mapping',
          completedSteps: new Set([...s.completedSteps, 'connect']),
        }))
      }
    } finally {
      setState((s) => ({ ...s, isLoadingSourceData: false }))
    }
  }
}

function completeStep(setState: SetState, step: WizardStep) {
  setState((s) => ({ ...s, completedSteps: new Set([...s.completedSteps, step]) }))
}

function createHandleMappingsComplete(
  setState: SetState,
  previewMutate: ReturnType<typeof usePreview>['mutate'],
) {
  return (newMappings: ImportMappings) => {
    setState((s) => ({ ...s, mappings: newMappings }))
    previewMutate(newMappings, {
      onSuccess: (previewData) => {
        setState((s) => ({
          ...s,
          preview: previewData,
          currentStep: 'preview',
          completedSteps: new Set([...s.completedSteps, 'mapping']),
        }))
      },
    })
  }
}

function createHandleStartImport(
  state: WizardState,
  setState: SetState,
  importMutate: ReturnType<typeof useExecuteImport>['mutate'],
) {
  return (selectedIds: number[]) => {
    if (!state.mappings) {
      return
    }
    const isMovie = state.sourceType === 'radarr'
    const mappingsWithSelection: ImportMappings = {
      ...state.mappings,
      ...(isMovie ? { selectedMovieTmdbIds: selectedIds } : { selectedSeriesTvdbIds: selectedIds }),
    }
    importMutate(mappingsWithSelection, {
      onSuccess: () => {
        completeStep(setState, 'preview')
        setState((s) => ({ ...s, currentStep: 'importing' }))
      },
    })
  }
}

export function useArrImportWizard() {
  const [state, setState] = useState<WizardState>(initialState)

  const { refetch: refetchRootFolders } = useSourceRootFolders()
  const { refetch: refetchQualityProfiles } = useSourceQualityProfiles()
  const previewMutation = usePreview()
  const executeImportMutation = useExecuteImport()
  const disconnectMutation = useDisconnect()

  const goToStep = (step: WizardStep) => {
    if (state.completedSteps.has(step)) {
      setState((s) => ({ ...s, currentStep: step }))
    }
  }

  return {
    ...state,
    setSourceType: (type: SourceType) => setState((s) => ({ ...s, sourceType: type })),
    isLoadingPreview: previewMutation.isPending,
    isImporting: executeImportMutation.isPending,
    handleConnected: createHandleConnected(setState, refetchRootFolders, refetchQualityProfiles),
    handleMappingsComplete: createHandleMappingsComplete(setState, previewMutation.mutate),
    handleStartImport: createHandleStartImport(state, setState, executeImportMutation.mutate),
    handleDone: () => disconnectMutation.mutate(undefined, { onSettled: () => setState(initialState) }),
    goToStep,
  }
}

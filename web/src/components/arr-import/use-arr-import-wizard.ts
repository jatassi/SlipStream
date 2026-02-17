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
}

const initialState: WizardState = {
  currentStep: 'connect',
  sourceType: 'radarr',
  mappings: null,
  preview: null,
  sourceRootFolders: [],
  sourceQualityProfiles: [],
  isLoadingSourceData: false,
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
        }))
      }
    } finally {
      setState((s) => ({ ...s, isLoadingSourceData: false }))
    }
  }
}

export function useArrImportWizard() {
  const [state, setState] = useState<WizardState>(initialState)

  const { refetch: refetchRootFolders } = useSourceRootFolders()
  const { refetch: refetchQualityProfiles } = useSourceQualityProfiles()
  const previewMutation = usePreview()
  const executeImportMutation = useExecuteImport()
  const disconnectMutation = useDisconnect()

  const handleConnected = createHandleConnected(setState, refetchRootFolders, refetchQualityProfiles)

  const handleMappingsComplete = (newMappings: ImportMappings) => {
    setState((s) => ({ ...s, mappings: newMappings }))
    previewMutation.mutate(newMappings, {
      onSuccess: (previewData) => {
        setState((s) => ({ ...s, preview: previewData, currentStep: 'preview' }))
      },
    })
  }

  const handleStartImport = () => {
    if (!state.mappings) {
      return
    }
    executeImportMutation.mutate(state.mappings, {
      onSuccess: () => setState((s) => ({ ...s, currentStep: 'importing' })),
    })
  }

  const handleDone = () => {
    disconnectMutation.mutate(undefined, {
      onSettled: () => setState(initialState),
    })
  }

  return {
    ...state,
    setSourceType: (type: SourceType) => setState((s) => ({ ...s, sourceType: type })),
    isLoadingPreview: previewMutation.isPending,
    isImporting: executeImportMutation.isPending,
    handleConnected,
    handleMappingsComplete,
    handleStartImport,
    handleDone,
  }
}

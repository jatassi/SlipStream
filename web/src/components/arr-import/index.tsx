import { Loader2 } from 'lucide-react'

import { ConnectStep } from './connect-step'
import { ImportStep } from './import-step'
import { MappingStep } from './mapping-step'
import { PreviewStep } from './preview-step'
import { useArrImportWizard } from './use-arr-import-wizard'

function LoadingState({ message }: { message: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <Loader2 className="text-muted-foreground mb-4 size-8 animate-spin" />
      <p className="text-muted-foreground text-sm">{message}</p>
    </div>
  )
}

export function ArrImportWizard() {
  const wizard = useArrImportWizard()

  return <WizardStepRenderer wizard={wizard} />
}

type WizardState = ReturnType<typeof useArrImportWizard>

function WizardStepRenderer({ wizard }: { wizard: WizardState }) {
  if (wizard.currentStep === 'connect') {
    return (
      <ConnectStep
        sourceType={wizard.sourceType}
        onSourceTypeChange={wizard.setSourceType}
        onConnected={wizard.handleConnected}
      />
    )
  }

  if (wizard.currentStep === 'mapping') {
    if (wizard.isLoadingSourceData) {
      return <LoadingState message="Loading source configuration..." />
    }

    return (
      <MappingStep
        sourceType={wizard.sourceType}
        sourceRootFolders={wizard.sourceRootFolders}
        sourceQualityProfiles={wizard.sourceQualityProfiles}
        onMappingsComplete={wizard.handleMappingsComplete}
      />
    )
  }

  if (wizard.currentStep === 'preview') {
    if (wizard.isLoadingPreview || !wizard.preview) {
      return <LoadingState message="Generating import preview..." />
    }

    return (
      <PreviewStep
        preview={wizard.preview}
        onStartImport={wizard.handleStartImport}
        isImporting={wizard.isImporting}
      />
    )
  }

  if (wizard.currentStep === 'importing') {
    return <ImportStep onDone={wizard.handleDone} />
  }

  return null
}

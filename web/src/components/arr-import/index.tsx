import { Check, Grid2x2Check, Import, Link2, Loader2, Plug } from 'lucide-react'

import type { WizardStep } from '@/hooks/use-arr-import'
import { cn } from '@/lib/utils'

import { ConnectStep } from './connect-step'
import { ImportStep } from './import-step'
import { MappingStep } from './mapping-step'
import { PreviewStep } from './preview-step'
import { useArrImportWizard } from './use-arr-import-wizard'

const WIZARD_STEPS: { step: WizardStep; label: string; icon: React.ElementType }[] = [
  { step: 'connect', label: 'Connect', icon: Plug },
  { step: 'mapping', label: 'Map Folders & Profiles', icon: Link2 },
  { step: 'preview', label: 'Select Items', icon: Grid2x2Check },
  { step: 'importing', label: 'Import', icon: Import },
]

function stepState(isActive: boolean, isCompleted: boolean) {
  if (isActive) {
    return 'active' as const
  }
  if (isCompleted) {
    return 'completed' as const
  }
  return 'incomplete' as const
}

const STEP_BUTTON_STYLES = {
  active: 'bg-primary/10 text-primary',
  completed: 'cursor-pointer text-green-500 hover:bg-muted',
  incomplete: 'cursor-default text-muted-foreground/50',
}

const STEP_CIRCLE_STYLES = {
  active: 'bg-primary text-primary-foreground',
  completed: 'bg-green-500/15 text-green-500',
  incomplete: 'bg-muted text-muted-foreground/50',
}

function WizardStepItem({
  label,
  icon: Icon,
  state,
  onClick,
}: {
  label: string
  icon: React.ElementType
  state: 'active' | 'completed' | 'incomplete'
  onClick?: () => void
}) {
  return (
    <button
      type="button"
      disabled={state !== 'completed'}
      onClick={onClick}
      className={cn(
        'flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors',
        STEP_BUTTON_STYLES[state],
      )}
    >
      <span
        className={cn(
          'flex size-6 shrink-0 items-center justify-center rounded-full',
          STEP_CIRCLE_STYLES[state],
        )}
      >
        {state === 'completed' ? <Check className="size-3.5" /> : <Icon className="size-3.5" />}
      </span>
      <span className="hidden sm:inline">{label}</span>
    </button>
  )
}

function WizardNav({
  currentStep,
  completedSteps,
  onStepClick,
}: {
  currentStep: WizardStep
  completedSteps: Set<WizardStep>
  onStepClick: (step: WizardStep) => void
}) {
  return (
    <nav className="flex items-center justify-center gap-1">
      {WIZARD_STEPS.map(({ step, label, icon }, i) => (
        <div key={step} className="flex items-center gap-1">
          <WizardStepItem
            label={label}
            icon={icon}
            state={stepState(currentStep === step, completedSteps.has(step))}
            onClick={completedSteps.has(step) && currentStep !== step ? () => onStepClick(step) : undefined}
          />
          {i < WIZARD_STEPS.length - 1 ? (
            <div className={cn('h-px w-6', completedSteps.has(step) ? 'bg-green-500/40' : 'bg-border')} />
          ) : null}
        </div>
      ))}
    </nav>
  )
}

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

  return (
    <>
      <WizardNav
        currentStep={wizard.currentStep}
        completedSteps={wizard.completedSteps}
        onStepClick={wizard.goToStep}
      />
      <div className="rounded-lg border border-border bg-card p-6">
        <WizardStepRenderer wizard={wizard} />
      </div>
    </>
  )
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
    if (wizard.isLoadingPreview || !wizard.preview || !wizard.mappings) {
      return <LoadingState message="Generating import preview..." />
    }

    return (
      <PreviewStep
        preview={wizard.preview}
        sourceType={wizard.sourceType}
        mappings={wizard.mappings}
        onStartImport={wizard.handleStartImport}
        isImporting={wizard.isImporting}
      />
    )
  }

  if (wizard.currentStep === 'importing') {
    return <ImportStep onDone={wizard.handleDone} sourceType={wizard.sourceType} />
  }

  return null
}

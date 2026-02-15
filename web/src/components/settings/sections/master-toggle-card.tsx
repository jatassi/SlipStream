import { FlaskConical, Info, Layers, X, XCircle } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { SlotNamingValidation } from '@/types'

import { ConfigValidationAlert } from './config-validation-alert'
import { NamingValidationAlert } from './naming-validation-alert'
import type { ValidationResult } from './use-version-slots-section'

function InfoBanner({ onDismiss }: { onDismiss: () => void }) {
  return (
    <Alert className="border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950/50">
      <Info className="size-4 text-blue-600 dark:text-blue-400" />
      <AlertTitle className="flex items-center justify-between">
        <span>Multi-Version Feature</span>
        <Button
          variant="ghost"
          size="icon"
          className="-mt-1 -mr-2 size-6"
          onClick={onDismiss}
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
  )
}

function MigrationErrorAlert({
  error,
  onDismiss,
}: {
  error: string
  onDismiss: () => void
}) {
  return (
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
          onClick={onDismiss}
        >
          <X className="size-4" />
        </Button>
      </AlertTitle>
      <AlertDescription className="text-red-700 dark:text-red-300">
        <p className="mt-1">{error}</p>
        <p className="mt-2 text-sm">
          Try running the dry run again to review your file assignments, or check that your slot
          configuration is valid.
        </p>
      </AlertDescription>
    </Alert>
  )
}

function DryRunAlert({
  configurationReady,
  onBegin,
}: {
  configurationReady: boolean
  onBegin: () => void
}) {
  return (
    <Alert className="border-purple-300 bg-purple-50 dark:border-purple-700 dark:bg-purple-950/50">
      <FlaskConical className="size-4 text-purple-600 dark:text-purple-400" />
      <div className="flex flex-1 items-center justify-between">
        <div>
          <AlertTitle>Dry Run</AlertTitle>
          <AlertDescription>See how your existing files will be organized</AlertDescription>
        </div>
        <Button onClick={onBegin} disabled={!configurationReady}>
          Begin
        </Button>
      </div>
    </Alert>
  )
}

function ToggleSwitch({
  multiVersionEnabled,
  isTogglePending,
  onToggle,
}: {
  multiVersionEnabled: boolean
  isTogglePending: boolean
  onToggle: (enabled: boolean) => void
}) {
  if (multiVersionEnabled) {
    return (
      <Switch
        id="multi-version-toggle"
        checked
        onCheckedChange={onToggle}
        disabled={isTogglePending}
        className="origin-right scale-150"
      />
    )
  }

  return (
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
  )
}

type SetupAlertsProps = {
  settingsEnabled: boolean
  migrationError: string | null
  configurationReady: boolean
  validationResult: ValidationResult
  namingValidation: SlotNamingValidation | null
  isValidatePending: boolean
  isValidateNamingPending: boolean
  onDismissMigrationError: () => void
  onBeginDryRun: () => void
  onValidate: () => void
  onValidateNaming: () => void
  onResolveConfig: () => void
  onResolveNaming: () => void
}

function SetupAlerts({
  settingsEnabled,
  migrationError,
  configurationReady,
  validationResult,
  namingValidation,
  isValidatePending,
  isValidateNamingPending,
  onDismissMigrationError,
  onBeginDryRun,
  onValidate,
  onValidateNaming,
  onResolveConfig,
  onResolveNaming,
}: SetupAlertsProps) {
  if (!migrationError && settingsEnabled) {
    return null
  }

  return (
    <CardContent className="space-y-3">
      {migrationError ? (
        <MigrationErrorAlert error={migrationError} onDismiss={onDismissMigrationError} />
      ) : null}

      {!settingsEnabled && (
        <DryRunAlert configurationReady={configurationReady} onBegin={onBeginDryRun} />
      )}

      {!settingsEnabled && (
        <ConfigValidationAlert
          validationResult={validationResult}
          configurationReady={configurationReady}
          isValidatePending={isValidatePending}
          onValidate={onValidate}
          onResolve={onResolveConfig}
        />
      )}

      {!settingsEnabled && (
        <NamingValidationAlert
          namingValidation={namingValidation}
          configurationReady={configurationReady}
          isValidateNamingPending={isValidateNamingPending}
          onValidateNaming={onValidateNaming}
          onResolve={onResolveNaming}
        />
      )}
    </CardContent>
  )
}

export type MasterToggleCardProps = {
  settingsEnabled: boolean
  multiVersionEnabled: boolean
  enabledSlotCount: number
  isTogglePending: boolean
  configurationReady: boolean
  migrationError: string | null
  infoCardDismissed: boolean
  validationResult: ValidationResult
  namingValidation: SlotNamingValidation | null
  isValidatePending: boolean
  isValidateNamingPending: boolean
  onToggleMultiVersion: (enabled: boolean) => void
  onDismissInfo: () => void
  onDismissMigrationError: () => void
  onBeginDryRun: () => void
  onValidate: () => void
  onValidateNaming: () => void
  onResolveConfig: () => void
  onResolveNaming: () => void
}

export function MasterToggleCard(props: MasterToggleCardProps) {
  const description = props.settingsEnabled
    ? `Active with ${props.enabledSlotCount} slot${props.enabledSlotCount === 1 ? '' : 's'} enabled`
    : 'Use the dry run below to preview and enable multi-version mode'

  return (
    <>
      {!props.infoCardDismissed && !props.multiVersionEnabled && (
        <InfoBanner onDismiss={props.onDismissInfo} />
      )}

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <CardTitle className="flex items-center gap-2">
                <Layers className="size-5" />
                Multi-Version Mode
              </CardTitle>
              <CardDescription>{description}</CardDescription>
            </div>
            <ToggleSwitch
              multiVersionEnabled={props.multiVersionEnabled}
              isTogglePending={props.isTogglePending}
              onToggle={props.onToggleMultiVersion}
            />
          </div>
        </CardHeader>
        <SetupAlerts
          settingsEnabled={props.settingsEnabled}
          migrationError={props.migrationError}
          configurationReady={props.configurationReady}
          validationResult={props.validationResult}
          namingValidation={props.namingValidation}
          isValidatePending={props.isValidatePending}
          isValidateNamingPending={props.isValidateNamingPending}
          onDismissMigrationError={props.onDismissMigrationError}
          onBeginDryRun={props.onBeginDryRun}
          onValidate={props.onValidate}
          onValidateNaming={props.onValidateNaming}
          onResolveConfig={props.onResolveConfig}
          onResolveNaming={props.onResolveNaming}
        />
      </Card>
    </>
  )
}

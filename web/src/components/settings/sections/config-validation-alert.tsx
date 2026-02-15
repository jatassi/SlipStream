import { AlertTriangle, Check, Settings, Wand2 } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

import type { ValidationResult } from './use-version-slots-section'

type ConfigValidationAlertProps = {
  validationResult: ValidationResult
  configurationReady: boolean
  isValidatePending: boolean
  onValidate: () => void
  onResolve: () => void
}

function getAlertClassName(result: ValidationResult): string {
  if (result === null) {
    return ''
  }
  if (result.valid) {
    return 'border-green-500 dark:border-green-600'
  }
  return 'border-orange-400 dark:border-orange-500'
}

function ValidationIcon({ result }: { result: ValidationResult }) {
  if (result === null) {
    return <Settings className="size-4" />
  }
  if (result.valid) {
    return <Check className="size-4 text-green-600 dark:text-green-400" />
  }
  return <AlertTriangle className="size-4 text-orange-500 dark:text-orange-400" />
}

function getStatusLabel(result: ValidationResult): string {
  if (result === null) {
    return 'Validation'
  }
  if (result.valid) {
    return 'Valid'
  }
  return 'Not Valid'
}

function ValidDescription() {
  return (
    <div className="mt-2">
      <span>Slot profiles are mutually exclusive based on one or more of:</span>
      <ul className="text-muted-foreground mt-1 ml-4 space-y-0.5 text-sm">
        <li>Different allowed quality tiers (e.g., 1080p vs 2160p)</li>
        <li>Conflicting HDR requirements (e.g., HDR required vs SDR required)</li>
        <li>Conflicting video codec requirements</li>
        <li>Conflicting audio codec or channel requirements</li>
      </ul>
    </div>
  )
}

function InvalidDescription({ validationResult }: { validationResult: ValidationResult }) {
  if (!validationResult) {
    return null
  }

  const nonConflictErrors = validationResult.errors?.filter(
    (e) => !e.startsWith('Profile conflict'),
  )
  const conflicts = validationResult.conflicts

  return (
    <div className="mt-2">
      {nonConflictErrors && nonConflictErrors.length > 0 ? (
        <ul className="list-inside list-disc">
          {nonConflictErrors.map((error) => (
            <li key={error}>{error}</li>
          ))}
        </ul>
      ) : null}
      {conflicts && conflicts.length > 0 ? (
        <div className="mt-2 space-y-3">
          {conflicts.map((conflict) => (
            <div key={`${conflict.slotAName}-${conflict.slotBName}`}>
              <p className="font-medium">
                Conflict between {conflict.slotAName} and {conflict.slotBName}:
              </p>
              <ul className="mt-1 ml-4 list-inside list-disc space-y-0.5">
                {conflict.issues.map((issue) => (
                  <li key={`${issue.attribute}-${issue.message}`}>
                    <span className="font-medium">{issue.attribute}:</span> {issue.message}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  )
}

function DescriptionContent({ validationResult }: { validationResult: ValidationResult }) {
  if (validationResult === null) {
    return <>Check that assigned Quality Profiles are mutually exclusive</>
  }
  if (validationResult.valid) {
    return <ValidDescription />
  }
  return <InvalidDescription validationResult={validationResult} />
}

export function ConfigValidationAlert({
  validationResult,
  configurationReady,
  isValidatePending,
  onValidate,
  onResolve,
}: ConfigValidationAlertProps) {
  const showResolveButton = validationResult !== null && !validationResult.valid

  return (
    <Alert className={getAlertClassName(validationResult)}>
      <ValidationIcon result={validationResult} />
      <div className="flex flex-1 items-center justify-between">
        <div className="flex-1">
          <AlertTitle>Quality Profiles {getStatusLabel(validationResult)}</AlertTitle>
          <AlertDescription>
            <DescriptionContent validationResult={validationResult} />
          </AlertDescription>
        </div>
        {showResolveButton ? (
          <Button
            onClick={onResolve}
            className="ml-4 shrink-0 bg-orange-500 text-white hover:bg-orange-600"
          >
            <Wand2 className="mr-2 size-4" />
            Resolve...
          </Button>
        ) : (
          <Button
            variant="outline"
            onClick={onValidate}
            disabled={!configurationReady || isValidatePending}
            className="ml-4 shrink-0"
          >
            {isValidatePending ? 'Validating...' : 'Validate'}
          </Button>
        )}
      </div>
    </Alert>
  )
}

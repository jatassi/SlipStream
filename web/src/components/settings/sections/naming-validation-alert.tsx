import { AlertTriangle, Check, FileText, Wand2 } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import type { SlotNamingValidation } from '@/types'

type NamingValidationAlertProps = {
  namingValidation: SlotNamingValidation | null
  configurationReady: boolean
  isValidateNamingPending: boolean
  onValidateNaming: () => void
  onResolve: () => void
}

function getAlertClassName(validation: SlotNamingValidation | null): string {
  if (validation === null) {
    return ''
  }
  if (validation.noEnabledSlots) {
    return 'border-orange-400 dark:border-orange-500'
  }
  if (validation.canProceed) {
    return 'border-green-500 dark:border-green-600'
  }
  return 'border-orange-400 dark:border-orange-500'
}

function NamingIcon({ validation }: { validation: SlotNamingValidation | null }) {
  if (validation === null) {
    return <FileText className="size-4" />
  }
  if (!validation.noEnabledSlots && validation.canProceed) {
    return <Check className="size-4 text-green-600 dark:text-green-400" />
  }
  return <AlertTriangle className="size-4 text-orange-500 dark:text-orange-400" />
}

function getStatusLabel(validation: SlotNamingValidation | null): string {
  if (validation === null) {
    return 'Validation'
  }
  if (validation.noEnabledSlots) {
    return 'Blocked'
  }
  if (validation.canProceed) {
    return 'Valid'
  }
  return 'Not Valid'
}

function CanProceedDescription({ validation }: { validation: SlotNamingValidation }) {
  if (validation.requiredAttributes.length === 0) {
    if (validation.qualityTierExclusive) {
      return (
        <p className="mt-2">
          Profiles are distinguished by quality tier - no additional filename tokens required
        </p>
      )
    }
    return <p className="mt-2">Complete Quality Profile validation first</p>
  }
  return (
    <p className="mt-2">
      File name formats include tokens for differentiating attributes:{' '}
      {validation.requiredAttributes.join(', ')}
    </p>
  )
}

function MissingTokensList({
  label,
  tokens,
}: {
  label: string
  tokens: { attribute: string; suggestedToken: string }[]
}) {
  return (
    <div>
      <p className="font-medium">{label}</p>
      <ul className="mt-1 ml-4 list-inside list-disc space-y-0.5">
        {tokens.map((token) => (
          <li key={`${token.attribute}-${token.suggestedToken}`}>
            <span className="font-medium">{token.attribute}:</span> Add{' '}
            <code className="bg-muted rounded px-1">{token.suggestedToken}</code>
          </li>
        ))}
      </ul>
    </div>
  )
}

function NotValidDescription({ validation }: { validation: SlotNamingValidation }) {
  return (
    <div className="mt-2 space-y-3">
      <p>
        Slots have different requirements for:{' '}
        <span className="font-medium">{validation.requiredAttributes.join(', ')}</span>
      </p>
      {!validation.movieFormatValid && validation.movieValidation.missingTokens ? (
        <MissingTokensList
          label="Movie filename format missing tokens:"
          tokens={validation.movieValidation.missingTokens}
        />
      ) : null}
      {!validation.episodeFormatValid && validation.episodeValidation.missingTokens ? (
        <MissingTokensList
          label="Episode filename format missing tokens:"
          tokens={validation.episodeValidation.missingTokens}
        />
      ) : null}
    </div>
  )
}

function NamingDescriptionContent({ validation }: { validation: SlotNamingValidation | null }) {
  if (validation === null) {
    return <>Verify filename formats include required differentiator tokens</>
  }
  if (validation.noEnabledSlots) {
    return (
      <p className="mt-2 text-orange-700 dark:text-orange-300">
        Complete Quality Profile validation first to ensure slot profiles are mutually exclusive.
      </p>
    )
  }
  if (validation.canProceed) {
    return <CanProceedDescription validation={validation} />
  }
  return <NotValidDescription validation={validation} />
}

export function NamingValidationAlert({
  namingValidation,
  configurationReady,
  isValidateNamingPending,
  onValidateNaming,
  onResolve,
}: NamingValidationAlertProps) {
  const showResolveButton = namingValidation !== null && !namingValidation.canProceed

  return (
    <Alert className={getAlertClassName(namingValidation)}>
      <NamingIcon validation={namingValidation} />
      <div className="flex flex-1 items-center justify-between">
        <div className="flex-1">
          <AlertTitle>File Naming {getStatusLabel(namingValidation)}</AlertTitle>
          <AlertDescription>
            <NamingDescriptionContent validation={namingValidation} />
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
            onClick={onValidateNaming}
            disabled={!configurationReady || isValidateNamingPending}
            className="ml-4 shrink-0"
          >
            {isValidateNamingPending ? 'Validating...' : 'Validate'}
          </Button>
        )}
      </div>
    </Alert>
  )
}

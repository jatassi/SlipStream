import { useEffect, useState } from 'react'

import { Pencil } from 'lucide-react'

import { Label } from '@/components/ui/label'
import { usePreviewNamingPattern } from '@/hooks'
import { useDebounce } from '@/hooks/use-debounce'

import { TokenBuilderDialog } from './token-builder-dialog'

const EMPTY_TOKENS: string[] = []

type PatternEditorProps = {
  label: string
  value: string
  onChange: (value: string) => void
  mediaType: 'episode' | 'movie'
  highlightTokens?: string[]
}

function PatternPreview({ valid, preview, error }: { valid: boolean; preview: string; error?: string }) {
  return (
    <div className="bg-muted/50 rounded-md p-2 text-xs">
      <span className="text-muted-foreground">Preview: </span>
      {valid ? (
        <span className="font-mono break-all text-green-600 dark:text-green-400">{preview}</span>
      ) : (
        <span className="text-red-600 dark:text-red-400">{error}</span>
      )}
    </div>
  )
}

function PatternTriggerButton({
  value,
  isMissing,
  onClick,
}: {
  value: string
  isMissing: boolean
  onClick: () => void
}) {
  const borderClass = isMissing ? 'border-orange-400' : ''
  return (
    <button
      type="button"
      onClick={onClick}
      className={`bg-muted/50 hover:bg-muted flex w-full cursor-pointer items-start gap-2 rounded-md border p-2 text-left font-mono text-xs transition-colors ${borderClass}`}
    >
      <Pencil className="text-muted-foreground mt-0.5 size-3 shrink-0" />
      <span className="break-all">{value || '(not configured)'}</span>
    </button>
  )
}

export function PatternEditor({
  label,
  value,
  onChange,
  mediaType,
  highlightTokens = EMPTY_TOKENS,
}: PatternEditorProps) {
  const [localValue, setLocalValue] = useState(value)
  const [prevValue, setPrevValue] = useState(value)
  const [tokenDialogOpen, setTokenDialogOpen] = useState(false)
  const debouncedValue = useDebounce(localValue, 500)
  const previewMutation = usePreviewNamingPattern()
  const previewMutate = previewMutation.mutate

  if (value !== prevValue) {
    setPrevValue(value)
    setLocalValue(value)
  }

  useEffect(() => {
    if (debouncedValue) {
      previewMutate({ pattern: debouncedValue, mediaType })
    }
  }, [debouncedValue, mediaType, previewMutate])

  const handleChange = (newValue: string) => {
    setLocalValue(newValue)
    onChange(newValue)
  }

  const preview = previewMutation.data
  const isMissingTokens = highlightTokens.some((token) => !localValue.includes(token))

  return (
    <div className="space-y-2">
      <Label className="text-xs">{label}</Label>
      <PatternTriggerButton
        value={localValue}
        isMissing={isMissingTokens}
        onClick={() => setTokenDialogOpen(true)}
      />
      {preview ? (
        <PatternPreview valid={preview.valid} preview={preview.preview} error={preview.error} />
      ) : null}
      <TokenBuilderDialog
        open={tokenDialogOpen}
        onOpenChange={setTokenDialogOpen}
        value={localValue}
        onChange={handleChange}
        mediaType={mediaType}
        highlightTokens={highlightTokens}
      />
    </div>
  )
}

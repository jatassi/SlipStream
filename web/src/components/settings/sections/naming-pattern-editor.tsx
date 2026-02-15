import { useEffect, useState } from 'react'

import { Pencil } from 'lucide-react'

import { Label } from '@/components/ui/label'
import { usePreviewNamingPattern } from '@/hooks'
import { useDebounce } from '@/hooks/use-debounce'
import type { TokenBreakdown } from '@/types'

import type { TokenContext } from './file-naming-constants'
import { TokenBuilderDialog } from './token-builder-dialog'

function PatternPreview({ preview }: { preview: { valid: boolean; preview: string; error?: string; tokens?: TokenBreakdown[] } }) {
  return (
    <div className="bg-muted/50 space-y-2 rounded-md p-3">
      <div className="flex items-start gap-2">
        <span className="shrink-0 text-xs font-medium">Preview:</span>
        {preview.valid ? (
          <span className="font-mono text-sm break-all text-green-600 dark:text-green-400">
            {preview.preview}
          </span>
        ) : (
          <span className="text-sm text-red-600 dark:text-red-400">{preview.error}</span>
        )}
      </div>
      {preview.tokens && preview.tokens.length > 0 ? (
        <details className="text-xs">
          <summary className="text-muted-foreground hover:text-foreground cursor-pointer">
            Token breakdown
          </summary>
          <div className="mt-2 space-y-1">
            {preview.tokens.map((t) => (
              <div key={`${t.token}-${t.value}`} className="flex items-center gap-2 font-mono">
                <span className="text-muted-foreground">{t.token}</span>
                <span className="text-muted-foreground">{'\u2192'}</span>
                <span className={t.empty ? 'text-yellow-600' : ''}>{t.value || '(empty)'}</span>
              </div>
            ))}
          </div>
        </details>
      ) : null}
    </div>
  )
}

type PatternEditorProps = {
  label: string
  value: string
  onChange: (value: string) => void
  description?: string
  mediaType?: 'episode' | 'movie' | 'folder'
  tokenContext: TokenContext
}

function usePatternPreview(value: string, mediaType: 'movie' | 'episode' | 'folder') {
  const [localValue, setLocalValue] = useState(value)
  const debouncedValue = useDebounce(localValue, 500)
  const previewMutation = usePreviewNamingPattern()
  const previewMutate = previewMutation.mutate

  useEffect(() => {
    setLocalValue(value)
  }, [value])

  useEffect(() => {
    if (debouncedValue) {
      previewMutate({ pattern: debouncedValue, mediaType })
    }
  }, [debouncedValue, mediaType, previewMutate])

  return { localValue, setLocalValue, preview: previewMutation.data }
}

export function PatternEditor({ label, value, onChange, description, mediaType = 'episode', tokenContext }: PatternEditorProps) {
  const { localValue, setLocalValue, preview } = usePatternPreview(value, mediaType)
  const [tokenDialogOpen, setTokenDialogOpen] = useState(false)

  const handleChange = (newValue: string) => {
    setLocalValue(newValue)
    onChange(newValue)
  }

  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <button
        type="button"
        onClick={() => setTokenDialogOpen(true)}
        className="bg-muted/50 hover:bg-muted flex w-full cursor-pointer items-start gap-3 rounded-md border p-3 text-left font-mono text-sm transition-colors"
      >
        <Pencil className="text-muted-foreground mt-0.5 size-4 shrink-0" />
        <span className="break-all">{localValue || '(not configured)'}</span>
      </button>
      {description ? <p className="text-muted-foreground text-xs">{description}</p> : null}
      {preview ? <PatternPreview preview={preview} /> : null}
      <TokenBuilderDialog
        open={tokenDialogOpen}
        onOpenChange={setTokenDialogOpen}
        value={localValue}
        onChange={handleChange}
        tokenContext={tokenContext}
      />
    </div>
  )
}

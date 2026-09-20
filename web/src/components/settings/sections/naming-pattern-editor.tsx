import { useEffect, useState } from 'react'

import { Pencil } from 'lucide-react'

import { Group } from '@/components/grouped-list'
import { useModuleNamingPreview, usePreviewNamingPattern } from '@/hooks'
import { useDebounce } from '@/hooks/use-debounce'
import type { TokenBreakdown, TokenContext as BackendTokenContext } from '@/types'

import { TokenBuilderDialog } from './token-builder-dialog'

function PatternPreview({ preview }: { preview: { valid: boolean; preview: string; error?: string; tokens?: TokenBreakdown[] } }) {
  return (
    <div className="space-y-2 px-4 py-3">
      <div className="flex items-start gap-2">
        <span className="text-footnote shrink-0 font-medium">Preview:</span>
        {preview.valid ? (
          <span className="font-mono text-footnote break-all text-green-600 dark:text-green-400">
            {preview.preview}
          </span>
        ) : (
          <span className="text-footnote text-red-600 dark:text-red-400">{preview.error}</span>
        )}
      </div>
      {preview.tokens && preview.tokens.length > 0 ? (
        <details className="text-caption">
          <summary className="text-muted-foreground cursor-pointer">Token breakdown</summary>
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
  mediaType?: string
  tokenContext: string
  moduleId?: string
  contextName?: string
  dynamicTokenContexts?: BackendTokenContext[]
}

type PreviewOptions = { value: string; mediaType: string; moduleId?: string; contextName?: string }

function usePatternPreview({ value, mediaType, moduleId, contextName }: PreviewOptions) {
  const [localValue, setLocalValue] = useState(value)
  const [prevValue, setPrevValue] = useState(value)
  const debouncedValue = useDebounce(localValue, 500)
  const legacyPreview = usePreviewNamingPattern()
  const modulePreview = useModuleNamingPreview(moduleId ?? '')
  const isModuleScoped = !!(moduleId && contextName)
  const previewMutation = isModuleScoped ? modulePreview : legacyPreview
  const previewMutate = previewMutation.mutate

  if (value !== prevValue) {
    setPrevValue(value)
    setLocalValue(value)
  }

  useEffect(() => {
    if (debouncedValue) {
      if (isModuleScoped) {
        previewMutate({ contextName, pattern: debouncedValue } as never)
      } else {
        previewMutate({ pattern: debouncedValue, mediaType } as never)
      }
    }
  }, [debouncedValue, mediaType, contextName, isModuleScoped, previewMutate])

  return { localValue, setLocalValue, preview: previewMutation.data }
}

export function PatternEditor({ label, value, onChange, description, mediaType = 'episode', tokenContext, moduleId, contextName, dynamicTokenContexts }: PatternEditorProps) {
  const { localValue, setLocalValue, preview } = usePatternPreview({ value, mediaType, moduleId, contextName })
  const [tokenDialogOpen, setTokenDialogOpen] = useState(false)

  const handleChange = (newValue: string) => {
    setLocalValue(newValue)
    onChange(newValue)
  }

  return (
    <Group header={label} footer={description}>
      <button
        type="button"
        aria-label={label}
        onClick={() => setTokenDialogOpen(true)}
        className="press-row flex min-h-tap w-full items-start gap-3 px-4 py-3 text-left font-mono text-footnote"
      >
        <Pencil className="text-muted-foreground mt-0.5 size-4 shrink-0" />
        <span className="break-all">{localValue || '(not configured)'}</span>
      </button>
      {preview ? <PatternPreview preview={preview} /> : null}
      <TokenBuilderDialog
        open={tokenDialogOpen}
        onOpenChange={setTokenDialogOpen}
        value={localValue}
        onChange={handleChange}
        tokenContext={tokenContext}
        dynamicTokenContexts={dynamicTokenContexts}
      />
    </Group>
  )
}

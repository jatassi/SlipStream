import type React from 'react'
import { useRef, useState } from 'react'

import { Code2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import { TOKEN_REFERENCE, type TokenCategory } from './resolve-naming-constants'

type TokenBuilderDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  value: string
  onChange: (value: string) => void
  mediaType: 'episode' | 'movie'
  highlightTokens: string[]
}

function TokenButton({
  token,
  description,
  example,
  highlighted,
  onInsert,
}: {
  token: string
  description: string
  example: string
  highlighted: boolean
  onInsert: (token: string) => void
}) {
  const base = highlighted
    ? 'border-orange-400 bg-orange-100 hover:bg-orange-200 dark:bg-orange-950 dark:hover:bg-orange-900'
    : 'bg-muted hover:bg-muted/80'

  return (
    <button
      type="button"
      onClick={() => onInsert(token)}
      className={`inline-flex cursor-pointer items-center gap-1 rounded border px-2 py-1 font-mono text-[10px] transition-colors ${base}`}
      title={`${description}\nExample: ${example}`}
    >
      <Code2 className="text-muted-foreground size-2.5" />
      {token}
    </button>
  )
}

function TokenCategorySection({
  category,
  highlightTokens,
  onInsert,
}: {
  category: TokenCategory
  highlightTokens: string[]
  onInsert: (token: string) => void
}) {
  const canHighlight = category === 'quality' || category === 'mediaInfo'

  return (
    <div className="space-y-1.5">
      <h4 className="text-muted-foreground text-xs font-medium capitalize">
        {category.replaceAll(/([A-Z])/g, ' $1').trim()}
      </h4>
      <div className="flex flex-wrap gap-1.5">
        {TOKEN_REFERENCE[category].map((t) => (
          <TokenButton
            key={t.token}
            token={t.token}
            description={t.description}
            example={t.example}
            highlighted={canHighlight ? highlightTokens.includes(t.token) : false}
            onInsert={onInsert}
          />
        ))}
      </div>
    </div>
  )
}

function PatternTextarea({
  textareaRef,
  value,
  onChange,
  onCursorUpdate,
}: {
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  value: string
  onChange: (value: string) => void
  onCursorUpdate: (e: React.SyntheticEvent<HTMLTextAreaElement>) => void
}) {
  return (
    <div className="space-y-2 border-t pt-2">
      <Label className="text-xs">Format Pattern</Label>
      <Textarea
        ref={textareaRef}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onSelect={onCursorUpdate}
        onClick={onCursorUpdate}
        onKeyUp={onCursorUpdate}
        className="min-h-[60px] font-mono text-xs"
        placeholder="Click tokens above to build your format pattern..."
      />
    </div>
  )
}

function useTokenBuilderState(open: boolean, value: string) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [localValue, setLocalValue] = useState(value)
  const [cursorPosition, setCursorPosition] = useState<number | null>(null)
  const [prevOpen, setPrevOpen] = useState(open)
  const [prevValue, setPrevValue] = useState(value)

  if (open !== prevOpen || value !== prevValue) {
    setPrevOpen(open)
    setPrevValue(value)
    if (open) {
      setLocalValue(value)
      setCursorPosition(null)
    }
  }

  const handleInsertToken = (token: string) => {
    const textarea = textareaRef.current
    if (!textarea) {
      setLocalValue((prev) => prev + token)
      return
    }
    const start = cursorPosition ?? textarea.selectionStart
    const newValue = localValue.slice(0, start) + token + localValue.slice(start)
    setLocalValue(newValue)
    const newPosition = start + token.length
    setCursorPosition(newPosition)
    setTimeout(() => {
      textarea.focus()
      textarea.setSelectionRange(newPosition, newPosition)
    }, 0)
  }

  const handleCursorUpdate = (e: React.SyntheticEvent<HTMLTextAreaElement>) => {
    setCursorPosition(e.currentTarget.selectionStart)
  }

  return { textareaRef, localValue, setLocalValue, handleInsertToken, handleCursorUpdate }
}

export function TokenBuilderDialog({
  open,
  onOpenChange,
  value,
  onChange,
  mediaType,
  highlightTokens,
}: TokenBuilderDialogProps) {
  const state = useTokenBuilderState(open, value)
  const categories: TokenCategory[] =
    mediaType === 'episode'
      ? ['episode', 'quality', 'mediaInfo']
      : ['movie', 'quality', 'mediaInfo']

  const handleApply = () => {
    onChange(state.localValue)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[70vh] flex-col sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Edit Pattern</DialogTitle>
          <DialogDescription>
            Click a token to insert it. Highlighted tokens are recommended.
          </DialogDescription>
        </DialogHeader>
        <div className="flex-1 space-y-3 overflow-y-auto py-2">
          {categories.map((category) => (
            <TokenCategorySection
              key={category}
              category={category}
              highlightTokens={highlightTokens}
              onInsert={state.handleInsertToken}
            />
          ))}
        </div>
        <PatternTextarea
          textareaRef={state.textareaRef}
          value={state.localValue}
          onChange={state.setLocalValue}
          onCursorUpdate={state.handleCursorUpdate}
        />
        <DialogFooter showCloseButton>
          <Button onClick={handleApply}>Apply</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

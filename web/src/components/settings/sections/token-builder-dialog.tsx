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

import type { TokenCategory, TokenContext } from './file-naming-constants'
import { TOKEN_CATEGORIES_BY_CONTEXT, TOKEN_REFERENCE } from './file-naming-constants'

function useTokenInserter(initialValue: string, open: boolean) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [localValue, setLocalValue] = useState(initialValue)
  const [cursorPosition, setCursorPosition] = useState<number | null>(null)
  const [prevOpen, setPrevOpen] = useState(open)
  const [prevValue, setPrevValue] = useState(initialValue)

  if (open !== prevOpen || initialValue !== prevValue) {
    setPrevOpen(open)
    setPrevValue(initialValue)
    if (open) {
      setLocalValue(initialValue)
      setCursorPosition(null)
    }
  }

  const insertToken = (token: string) => {
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

  return { textareaRef, localValue, setLocalValue, insertToken, handleCursorUpdate }
}

function TokenCategoryList({
  categories,
  onInsert,
}: {
  categories: readonly TokenCategory[]
  onInsert: (token: string) => void
}) {
  return (
    <div className="flex-1 space-y-4 overflow-y-auto py-2">
      {categories.map((category) => (
        <div key={category} className="space-y-2">
          <h4 className="text-muted-foreground text-sm font-medium capitalize">
            {category.replaceAll(/([A-Z])/g, ' $1').trim()}
          </h4>
          <div className="flex flex-wrap gap-2">
            {TOKEN_REFERENCE[category].map((t) => (
              <button
                key={t.token}
                type="button"
                onClick={() => onInsert(t.token)}
                className="bg-muted hover:bg-muted/80 inline-flex cursor-pointer items-center gap-1.5 rounded-md border px-2.5 py-1.5 font-mono text-xs transition-colors"
                title={`${t.description}\nExample: ${t.example}`}
              >
                <Code2 className="text-muted-foreground size-3" />
                {t.token}
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

export function TokenBuilderDialog({
  open,
  onOpenChange,
  value,
  onChange,
  tokenContext,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  value: string
  onChange: (value: string) => void
  tokenContext: TokenContext
}) {
  const { textareaRef, localValue, setLocalValue, insertToken, handleCursorUpdate } =
    useTokenInserter(value, open)

  const handleApply = () => {
    onChange(localValue)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[80vh] flex-col sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Token Builder</DialogTitle>
          <DialogDescription>Click a token to insert it into your format pattern</DialogDescription>
        </DialogHeader>
        <TokenCategoryList categories={TOKEN_CATEGORIES_BY_CONTEXT[tokenContext]} onInsert={insertToken} />
        <div className="space-y-2 border-t pt-2">
          <Label>Format Pattern</Label>
          <Textarea
            ref={textareaRef}
            value={localValue}
            onChange={(e) => setLocalValue(e.target.value)}
            onSelect={handleCursorUpdate}
            onClick={handleCursorUpdate}
            onKeyUp={handleCursorUpdate}
            className="min-h-[80px] font-mono text-sm"
            placeholder="Click tokens above to build your format pattern..."
          />
        </div>
        <DialogFooter showCloseButton>
          <Button onClick={handleApply}>Apply</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

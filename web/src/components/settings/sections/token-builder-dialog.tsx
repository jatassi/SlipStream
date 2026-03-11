import { useRef, useState } from 'react'

import { Code2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import type { TokenContext as BackendTokenContext } from '@/types'

import type { StaticTokenContext, TokenCategory } from './file-naming-constants'
import { TOKEN_CATEGORIES_BY_CONTEXT, TOKEN_REFERENCE } from './file-naming-constants'

type TokenEntry = { token: string; description: string; example: string }

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

function TokenButton({ entry, onInsert }: { entry: TokenEntry; onInsert: (token: string) => void }) {
  return (
    <button
      type="button"
      onClick={() => onInsert(entry.token)}
      className="bg-muted hover:bg-muted/80 inline-flex cursor-pointer items-center gap-1.5 rounded-md border px-2.5 py-1.5 font-mono text-xs transition-colors"
      title={`${entry.description}\nExample: ${entry.example}`}
    >
      <Code2 className="text-muted-foreground size-3" />
      {entry.token}
    </button>
  )
}

function StaticTokenCategoryList({
  categories,
  onInsert,
}: {
  categories: readonly TokenCategory[]
  onInsert: (token: string) => void
}) {
  return (
    <div className="space-y-4 py-2">
      {categories.map((category) => (
        <div key={category} className="space-y-2">
          <h4 className="text-muted-foreground text-sm font-medium capitalize">
            {category.replaceAll(/([A-Z])/g, ' $1').trim()}
          </h4>
          <div className="flex flex-wrap gap-2">
            {TOKEN_REFERENCE[category].map((t) => (
              <TokenButton key={t.token} entry={t} onInsert={onInsert} />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function DynamicTokenContextList({
  contexts,
  contextName,
  onInsert,
}: {
  contexts: BackendTokenContext[]
  contextName: string
  onInsert: (token: string) => void
}) {
  const matched = contexts.find((c) => c.name === contextName)
  if (!matched) { return null }

  const grouped = new Map<string, TokenEntry[]>()
  for (const t of matched.tokens) {
    const groupName = t.name.split(' ')[0] ?? 'Other'
    const existing = grouped.get(groupName) ?? []
    existing.push({ token: t.token, description: t.description, example: t.example })
    grouped.set(groupName, existing)
  }

  return (
    <div className="space-y-4 py-2">
      {[...grouped.entries()].map(([group, tokens]) => (
        <div key={group} className="space-y-2">
          <h4 className="text-muted-foreground text-sm font-medium capitalize">{group}</h4>
          <div className="flex flex-wrap gap-2">
            {tokens.map((t) => (
              <TokenButton key={t.token} entry={t} onInsert={onInsert} />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function isStaticContext(ctx: string): ctx is StaticTokenContext {
  return ctx in TOKEN_CATEGORIES_BY_CONTEXT
}

function TokenListContent({
  tokenContext,
  dynamicTokenContexts,
  onInsert,
}: {
  tokenContext: string
  dynamicTokenContexts?: BackendTokenContext[]
  onInsert: (token: string) => void
}) {
  if (dynamicTokenContexts?.some((c) => c.name === tokenContext)) {
    return (
      <DynamicTokenContextList
        contexts={dynamicTokenContexts}
        contextName={tokenContext}
        onInsert={onInsert}
      />
    )
  }

  if (isStaticContext(tokenContext)) {
    return <StaticTokenCategoryList categories={TOKEN_CATEGORIES_BY_CONTEXT[tokenContext]} onInsert={onInsert} />
  }

  return null
}

type TokenBuilderDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  value: string
  onChange: (value: string) => void
  tokenContext: string
  dynamicTokenContexts?: BackendTokenContext[]
}

function PatternInput({ textareaRef, value, onChange, onCursorUpdate }: {
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  value: string
  onChange: (value: string) => void
  onCursorUpdate: (e: React.SyntheticEvent<HTMLTextAreaElement>) => void
}) {
  return (
    <div className="shrink-0 space-y-2 border-t pt-2">
      <Label>Format Pattern</Label>
      <Textarea ref={textareaRef} value={value} onChange={(e) => onChange(e.target.value)} onSelect={onCursorUpdate} onClick={onCursorUpdate} onKeyUp={onCursorUpdate} className="min-h-[80px] font-mono text-sm" placeholder="Click tokens above to build your format pattern..." />
    </div>
  )
}

export function TokenBuilderDialog({ open, onOpenChange, value, onChange, tokenContext, dynamicTokenContexts }: TokenBuilderDialogProps) {
  const { textareaRef, localValue, setLocalValue, insertToken, handleCursorUpdate } = useTokenInserter(value, open)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[80vh] sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Token Builder</DialogTitle>
          <DialogDescription>Click a token to insert it into your format pattern</DialogDescription>
        </DialogHeader>
        <DialogBody>
          <TokenListContent tokenContext={tokenContext} dynamicTokenContexts={dynamicTokenContexts} onInsert={insertToken} />
        </DialogBody>
        <PatternInput textareaRef={textareaRef} value={localValue} onChange={setLocalValue} onCursorUpdate={handleCursorUpdate} />
        <DialogFooter showCloseButton>
          <Button onClick={() => { onChange(localValue); onOpenChange(false) }}>Apply</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

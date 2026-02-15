import { TOKEN_REFERENCE } from './resolve-naming-constants'

type TokenReferenceListProps = {
  stillMissingTokens: Set<string>
}

export function TokenReferenceList({ stillMissingTokens }: TokenReferenceListProps) {
  const tokens = [...TOKEN_REFERENCE.quality, ...TOKEN_REFERENCE.mediaInfo.slice(0, 5)]

  return (
    <div className="mt-6 space-y-3">
      <h4 className="text-muted-foreground text-xs font-medium">Differentiator Tokens</h4>
      <div className="grid gap-2">
        {tokens.map((t) => {
          const highlighted = stillMissingTokens.has(t.token)
          const style = highlighted
            ? 'border-orange-400 bg-orange-50 dark:bg-orange-950/30'
            : 'bg-muted/30'

          return (
            <div
              key={t.token}
              className={`flex items-center justify-between rounded border p-2 text-xs ${style}`}
            >
              <code className="font-mono">{t.token}</code>
              <span className="text-muted-foreground">{t.example}</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

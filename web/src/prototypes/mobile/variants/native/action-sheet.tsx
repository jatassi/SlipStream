import { useState } from 'react'

import { cn } from '@/lib/utils'

type Action = { label: string; onClick: () => void; destructive?: boolean }

type ActionSheetProps = {
  title: string
  subtitle?: string
  actions: Action[]
  onClose: () => void
}

export function ActionSheet({ title, subtitle, actions, onClose }: ActionSheetProps) {
  const [closing, setClosing] = useState(false)

  const dismiss = () => setClosing(true)
  const run = (action: Action) => {
    action.onClick()
    dismiss()
  }

  return (
    <div className="absolute inset-0 z-40" data-closing={closing ? '' : undefined}>
      <button type="button" aria-label="Dismiss" className="native-scrim w-full" onClick={dismiss} />
      <div
        className="native-sheet"
        onAnimationEnd={(e) => {
          if (closing && e.target === e.currentTarget) {
            onClose()
          }
        }}
      >
        <div className="m-material-heavy overflow-hidden rounded-[14px]">
          <div className="px-4 pt-3.5 pb-3 text-center">
            <div className="text-m-footnote font-semibold">{title}</div>
            {subtitle !== undefined && <div className="mt-0.5 truncate font-mono text-[11px] text-muted-foreground">{subtitle}</div>}
          </div>
          {actions.map((action) => (
            <button
              key={action.label}
              type="button"
              onClick={() => run(action)}
              className={cn(
                'm-press-row block h-14 w-full border-t border-border/70 text-m-heading font-normal tracking-normal',
                action.destructive ? 'text-red-400' : 'text-tv-400',
              )}
            >
              {action.label}
            </button>
          ))}
        </div>
        <button
          type="button"
          onClick={dismiss}
          className="m-press mt-2 block h-14 w-full rounded-[14px] bg-card text-m-heading font-semibold tracking-normal text-tv-400"
        >
          Cancel
        </button>
      </div>
    </div>
  )
}

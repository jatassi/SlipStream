import { Plus } from 'lucide-react'

import { cn } from '@/lib/utils'

type AddPlaceholderCardProps = {
  onClick: () => void
  label: string
  icon?: React.ReactNode
  className?: string
}

export function AddPlaceholderCard({ onClick, label, icon, className }: AddPlaceholderCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'border-muted-foreground/25 bg-muted/30 w-full rounded-xl border-2 border-dashed',
        'flex items-center justify-center gap-4 px-4 py-8',
        'text-muted-foreground text-sm',
        'hover:border-muted-foreground/40 hover:bg-muted/50 hover:text-foreground transition-colors',
        'focus-visible:ring-ring focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none',
        className,
      )}
    >
      <div className="bg-muted-foreground/10 flex size-10 items-center justify-center rounded-lg">
        {icon || <Plus className="size-5" />}
      </div>
      <span>{label}</span>
    </button>
  )
}

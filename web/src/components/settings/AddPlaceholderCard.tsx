import { Plus } from 'lucide-react'
import { cn } from '@/lib/utils'

interface AddPlaceholderCardProps {
  onClick: () => void
  label: string
  icon?: React.ReactNode
  className?: string
}

export function AddPlaceholderCard({
  onClick,
  label,
  icon,
  className,
}: AddPlaceholderCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'w-full rounded-xl border-2 border-dashed border-muted-foreground/25 bg-muted/30',
        'flex items-center justify-center gap-4 px-4 py-8',
        'text-sm text-muted-foreground',
        'transition-colors hover:border-muted-foreground/40 hover:bg-muted/50 hover:text-foreground',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        className
      )}
    >
      <div className="flex size-10 items-center justify-center rounded-lg bg-muted-foreground/10">
        {icon || <Plus className="size-5" />}
      </div>
      <span>{label}</span>
    </button>
  )
}

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type EmptyStateProps = {
  icon?: React.ReactNode
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
  }
  className?: string
}

export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn('flex flex-col items-center justify-center px-4 py-12 text-center', className)}
    >
      {icon ? <div className="text-muted-foreground mb-4">{icon}</div> : null}
      <h3 className="mb-1 text-lg font-semibold">{title}</h3>
      {description ? (
        <p className="text-muted-foreground mb-4 max-w-md text-sm">{description}</p>
      ) : null}
      {action ? <Button onClick={action.onClick}>{action.label}</Button> : null}
    </div>
  )
}

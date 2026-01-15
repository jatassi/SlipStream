import type { SummaryCardProps } from './types'

const variantStyles = {
  default: '',
  success: 'text-green-600 dark:text-green-400',
  warning: 'text-orange-600 dark:text-orange-400',
  error: 'text-red-600 dark:text-red-400',
}

export function SummaryCard({ label, value, icon: Icon, variant = 'default', active = false, onClick }: SummaryCardProps) {
  return (
    <div
      className={`border rounded-lg p-3 cursor-pointer transition-colors hover:bg-muted/50 ${
        active ? 'ring-2 ring-primary bg-muted/30' : ''
      }`}
      onClick={onClick}
    >
      <div className="flex items-center gap-2 text-muted-foreground text-sm">
        <Icon className={`size-4 ${variantStyles[variant]}`} />
        {label}
      </div>
      <div className={`text-2xl font-bold mt-1 ${variantStyles[variant]}`}>{value}</div>
    </div>
  )
}

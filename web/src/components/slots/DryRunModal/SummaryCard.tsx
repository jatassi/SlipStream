import type { SummaryCardProps } from './types'

const variantStyles = {
  default: '',
  success: 'text-green-600 dark:text-green-400',
  warning: 'text-orange-600 dark:text-orange-400',
  error: 'text-red-600 dark:text-red-400',
}

export function SummaryCard({
  label,
  value,
  icon: Icon,
  variant = 'default',
  active = false,
  onClick,
}: SummaryCardProps) {
  return (
    <div
      className={`hover:bg-muted/50 cursor-pointer rounded-lg border p-3 transition-colors ${
        active ? 'ring-primary bg-muted/30 ring-2' : ''
      }`}
      onClick={onClick}
    >
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Icon className={`size-4 ${variantStyles[variant]}`} />
        {label}
      </div>
      <div className={`mt-1 text-2xl font-bold ${variantStyles[variant]}`}>{value}</div>
    </div>
  )
}

import { cn } from '@/lib/utils'

export type ChipOption<T extends string> = {
  value: T
  label: string
}

export function Chip({
  label,
  active,
  onClick,
}: {
  label: string
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        'press text-footnote focus-visible:ring-ring inline-flex h-8 shrink-0 items-center rounded-full px-3 font-medium whitespace-nowrap outline-none focus-visible:ring-[3px]',
        active ? 'bg-foreground text-background' : 'bg-foreground/8 text-foreground/80',
      )}
    >
      {label}
    </button>
  )
}

type ChipRowProps<T extends string> = {
  label: string
  options: ChipOption<T>[]
  selected: T[]
  onToggle: (value: T) => void
  className?: string
}

export function ChipRow<T extends string>({
  label,
  options,
  selected,
  onToggle,
  className,
}: ChipRowProps<T>) {
  return (
    <div
      role="group"
      aria-label={label}
      className={cn('scroll-x px-screen flex items-center gap-2 pb-4', className)}
    >
      {options.map((option) => (
        <Chip
          key={option.value}
          label={option.label}
          active={selected.includes(option.value)}
          onClick={() => {
            onToggle(option.value)
          }}
        />
      ))}
    </div>
  )
}

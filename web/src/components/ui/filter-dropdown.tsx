import type { LucideIcon } from 'lucide-react'
import { ChevronDown, Filter } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'

type FilterTheme = 'movie' | 'tv' | 'neutral'

const THEME_ACTIVE_CLASS: Record<FilterTheme, string> = {
  movie: 'text-movie-400',
  tv: 'text-tv-400',
  neutral: 'text-white',
}

interface FilterOption<T extends string> {
  value: T
  label: string
  icon?: LucideIcon
}

interface FilterDropdownProps<T extends string> {
  options: FilterOption<T>[]
  selected: T[]
  onToggle: (value: T) => void
  onReset: () => void
  icon?: LucideIcon
  label?: string
  theme?: FilterTheme
  className?: string
  disabled?: boolean
}

export function FilterDropdown<T extends string>({
  options,
  selected,
  onToggle,
  onReset,
  icon: Icon = Filter,
  label = 'Items',
  theme = 'neutral',
  className,
  disabled,
}: FilterDropdownProps<T>) {
  const allSelected = selected.length >= options.length

  function getDisplayLabel(): string {
    if (allSelected) return `All ${label}`
    if (selected.length === 0) return `No ${label}`
    if (selected.length > 2) return `${selected.length} Selected`
    return options
      .filter((o) => selected.includes(o.value))
      .map((o) => o.label)
      .join(', ')
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          'border-input dark:bg-input/30 dark:hover:bg-input/50 focus-visible:border-ring focus-visible:ring-ring/50 gap-1.5 rounded-lg border bg-transparent py-2 pr-2 pl-2.5 text-sm transition-colors select-none focus-visible:ring-[3px] h-8 flex w-fit items-center whitespace-nowrap outline-none',
          disabled && 'pointer-events-none opacity-50',
          className,
        )}
      >
        <Icon className={cn('size-4 shrink-0', !allSelected ? THEME_ACTIVE_CLASS[theme] : 'text-muted-foreground')} />
        {getDisplayLabel()}
        <ChevronDown className="size-4 shrink-0 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-auto min-w-(--anchor-width)">
        {!allSelected && (
          <>
            <DropdownMenuItem onClick={onReset} className="text-muted-foreground">
              Reset
            </DropdownMenuItem>
            <DropdownMenuSeparator />
          </>
        )}
        {options.map((opt) => (
          <DropdownMenuCheckboxItem
            key={opt.value}
            checked={selected.includes(opt.value)}
            onCheckedChange={() => onToggle(opt.value)}
          >
            {opt.icon && <opt.icon className="size-4 shrink-0 text-muted-foreground" />}
            {opt.label}
          </DropdownMenuCheckboxItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

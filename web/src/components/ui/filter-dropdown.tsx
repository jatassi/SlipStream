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

type FilterOption<T extends string> = {
  value: T
  label: string
  icon?: LucideIcon
}

type FilterDropdownProps<T extends string> = {
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

type TriggerIconProps = {
  Icon: LucideIcon
  allSelected: boolean
  theme: FilterTheme
}

function TriggerIcon({ Icon, allSelected, theme }: TriggerIconProps) {
  return (
    <Icon
      className={cn(
        'size-4 shrink-0',
        allSelected ? 'text-muted-foreground' : THEME_ACTIVE_CLASS[theme],
      )}
    />
  )
}

type ResetItemProps = {
  onReset: () => void
}

function ResetItem({ onReset }: ResetItemProps) {
  return (
    <>
      <DropdownMenuItem onClick={onReset} className="text-muted-foreground">
        Reset
      </DropdownMenuItem>
      <DropdownMenuSeparator />
    </>
  )
}

type OptionItemProps<T extends string> = {
  option: FilterOption<T>
  checked: boolean
  onToggle: (value: T) => void
}

function OptionItem<T extends string>({ option, checked, onToggle }: OptionItemProps<T>) {
  return (
    <DropdownMenuCheckboxItem checked={checked} onCheckedChange={() => onToggle(option.value)}>
      {option.icon ? <option.icon className="text-muted-foreground size-4 shrink-0" /> : null}
      {option.label}
    </DropdownMenuCheckboxItem>
  )
}

type DisplayLabelParams<T extends string> = {
  options: FilterOption<T>[]
  selected: T[]
  label: string
  allSelected: boolean
}

function getDisplayLabel<T extends string>({
  options,
  selected,
  label,
  allSelected,
}: DisplayLabelParams<T>): string {
  if (allSelected) {
    return `All ${label}`
  }
  if (selected.length === 0) {
    return `No ${label}`
  }
  if (selected.length > 2) {
    return `${selected.length} Selected`
  }
  return options
    .filter((o) => selected.includes(o.value))
    .map((o) => o.label)
    .join(', ')
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
  const displayLabel = getDisplayLabel({ options, selected, label, allSelected })

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          'border-input dark:bg-input/30 dark:hover:bg-input/50 focus-visible:border-ring focus-visible:ring-ring/50 flex h-8 w-fit items-center gap-1.5 rounded-lg border bg-transparent py-2 pr-2 pl-2.5 text-sm whitespace-nowrap transition-colors outline-none select-none focus-visible:ring-[3px]',
          disabled && 'pointer-events-none opacity-50',
          className,
        )}
      >
        <TriggerIcon Icon={Icon} allSelected={allSelected} theme={theme} />
        {displayLabel}
        <ChevronDown className="text-muted-foreground size-4 shrink-0" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-auto min-w-(--anchor-width)">
        {!allSelected && <ResetItem onReset={onReset} />}
        {options.map((opt) => (
          <OptionItem
            key={opt.value}
            option={opt}
            checked={selected.includes(opt.value)}
            onToggle={onToggle}
          />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

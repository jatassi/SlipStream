import type { ComponentProps, ReactNode } from 'react'
import { useId } from 'react'

import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Slider } from '@/components/ui/slider'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

const ROW = 'flex min-h-tap w-full items-center gap-3 px-4 py-2'
const STACKED = 'flex min-h-tap w-full flex-col justify-center gap-2 px-4 py-2.5'
const LABEL = 'text-body min-w-0 flex-1 font-medium'
const FIELD = 'h-11 text-base'
const TRIGGER = 'h-11 data-[size=default]:h-11 text-base'

export type SelectOption = { value: string; label: string }

export function ControlRow({
  label,
  htmlFor,
  trailing,
  className,
}: {
  label: ReactNode
  htmlFor?: string
  trailing: ReactNode
  className?: string
}) {
  return (
    <div className={cn(ROW, className)}>
      {htmlFor === undefined ? (
        <span className={LABEL}>{label}</span>
      ) : (
        <label htmlFor={htmlFor} className={LABEL}>
          {label}
        </label>
      )}
      <div className="flex shrink-0 items-center gap-2">{trailing}</div>
    </div>
  )
}

export function StackedRow({
  label,
  value,
  children,
  className,
}: {
  label?: ReactNode
  value?: ReactNode
  children: ReactNode
  className?: string
}) {
  return (
    <div className={cn(STACKED, className)}>
      {label !== undefined && (
        <div className="flex items-baseline justify-between gap-3">
          <span className={LABEL}>{label}</span>
          {value !== undefined && (
            <span className="text-footnote shrink-0 text-muted-foreground">{value}</span>
          )}
        </div>
      )}
      {children}
    </div>
  )
}

export function ControlStack({
  footer,
  children,
  className,
}: {
  footer?: ReactNode
  children: ReactNode
  className?: string
}) {
  return (
    <div className={className}>
      <div className="divide-y divide-border/70 overflow-hidden rounded-card border">{children}</div>
      {footer !== undefined && (
        <div className="text-footnote px-3 pt-2 text-muted-foreground">{footer}</div>
      )}
    </div>
  )
}

export function SwitchRow({
  label,
  description,
  checked,
  onCheckedChange,
  disabled,
}: {
  label: string
  description?: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  disabled?: boolean
}) {
  return (
    <ControlRow
      label={
        description === undefined ? (
          label
        ) : (
          <span className="block">
            {label}
            <span className="text-footnote block font-normal text-muted-foreground">
              {description}
            </span>
          </span>
        )
      }
      trailing={
        <Switch
          aria-label={label}
          checked={checked}
          onCheckedChange={onCheckedChange}
          disabled={disabled}
        />
      }
    />
  )
}

export function SelectRow({
  label,
  value,
  onChange,
  options,
  disabled,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  options: readonly SelectOption[]
  disabled?: boolean
}) {
  return (
    <ControlRow
      label={label}
      trailing={
        <Select value={value} onValueChange={(next: string | null) => next !== null && onChange(next)}>
          <SelectTrigger aria-label={label} disabled={disabled} className={TRIGGER}>
            {options.find((option) => option.value === value)?.label ?? value}
          </SelectTrigger>
          <SelectContent>
            {options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      }
    />
  )
}

type InputRowProps = Omit<ComponentProps<'input'>, 'id'> & {
  label: string
  stacked?: boolean
  inputClassName?: string
}

export function InputRow({ label, stacked = false, inputClassName, ...props }: InputRowProps) {
  const id = useId()
  const field = <Input id={id} className={cn(FIELD, inputClassName)} {...props} />

  if (stacked) {
    return (
      <StackedRow>
        <label htmlFor={id} className={LABEL}>
          {label}
        </label>
        {field}
      </StackedRow>
    )
  }

  return <ControlRow label={label} htmlFor={id} trailing={<div className="w-28">{field}</div>} />
}

type TextareaRowProps = Omit<ComponentProps<'textarea'>, 'id'> & { label: string }

export function TextareaRow({ label, className, ...props }: TextareaRowProps) {
  const id = useId()
  return (
    <StackedRow>
      <label htmlFor={id} className={LABEL}>
        {label}
      </label>
      <Textarea id={id} className={cn('text-base', className)} {...props} />
    </StackedRow>
  )
}

export function SliderRow({
  label,
  value,
  display,
  onChange,
  min,
  max,
  step,
  disabled,
}: {
  label: string
  value: number
  display: string
  onChange: (value: number) => void
  min: number
  max: number
  step: number
  disabled?: boolean
}) {
  return (
    <StackedRow label={label} value={display}>
      <Slider
        label={label}
        value={[value]}
        onValueChange={(next) =>
          onChange(Array.isArray(next) && typeof next[0] === 'number' ? next[0] : value)
        }
        min={min}
        max={max}
        step={step}
        disabled={disabled}
      />
    </StackedRow>
  )
}

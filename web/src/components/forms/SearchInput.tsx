import { useCallback, useEffect, useState } from 'react'

import { Search, X } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

type SearchInputProps = {
  value?: string
  onChange: (value: string) => void
  placeholder?: string
  debounceMs?: number
  className?: string
  autoFocus?: boolean
}

export function SearchInput({
  value: controlledValue,
  onChange,
  placeholder = 'Search...',
  debounceMs = 900,
  className,
  autoFocus = false,
}: SearchInputProps) {
  const [internalValue, setInternalValue] = useState(controlledValue || '')
  const [prevControlledValue, setPrevControlledValue] = useState(controlledValue)

  // Sync with controlled value during render (React-recommended pattern)
  if (controlledValue !== undefined && controlledValue !== prevControlledValue) {
    setPrevControlledValue(controlledValue)
    setInternalValue(controlledValue)
  }

  // Debounce the onChange callback
  useEffect(() => {
    const timer = setTimeout(() => {
      onChange(internalValue)
    }, debounceMs)

    return () => clearTimeout(timer)
  }, [internalValue, debounceMs, onChange])

  const handleClear = useCallback(() => {
    setInternalValue('')
    onChange('')
  }, [onChange])

  return (
    <div className={cn('relative', className)}>
      <Search className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
      <Input
        type="search"
        value={internalValue}
        onChange={(e) => setInternalValue(e.target.value)}
        placeholder={placeholder}
        className="pr-9 pl-9"
        autoFocus={autoFocus}
      />
      {internalValue ? (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          onClick={handleClear}
          className="absolute top-1/2 right-1 size-7 -translate-y-1/2"
        >
          <X className="size-4" />
        </Button>
      ) : null}
    </div>
  )
}

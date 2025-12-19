import { useState, useEffect, useCallback } from 'react'
import { Search, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

interface SearchInputProps {
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
  debounceMs = 300,
  className,
  autoFocus = false,
}: SearchInputProps) {
  const [internalValue, setInternalValue] = useState(controlledValue || '')

  // Sync with controlled value
  useEffect(() => {
    if (controlledValue !== undefined) {
      setInternalValue(controlledValue)
    }
  }, [controlledValue])

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
      <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
      <Input
        type="search"
        value={internalValue}
        onChange={(e) => setInternalValue(e.target.value)}
        placeholder={placeholder}
        className="pl-9 pr-9"
        autoFocus={autoFocus}
      />
      {internalValue && (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          onClick={handleClear}
          className="absolute right-1 top-1/2 size-7 -translate-y-1/2"
        >
          <X className="size-4" />
        </Button>
      )}
    </div>
  )
}

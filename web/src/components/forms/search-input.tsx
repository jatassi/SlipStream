import { useEffect, useRef, useState } from 'react'

import { Search, X } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

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
  const [internalValue, setInternalValue] = useState(controlledValue ?? '')
  const [prevControlledValue, setPrevControlledValue] = useState(controlledValue)
  const inputRef = useRef<HTMLInputElement>(null)

  if (controlledValue !== undefined && controlledValue !== prevControlledValue) {
    setPrevControlledValue(controlledValue)
    setInternalValue(controlledValue)
  }
  useEffect(() => {
    if (autoFocus) {
      inputRef.current?.focus()
    }
    const timer = setTimeout(() => onChange(internalValue), debounceMs)
    return () => clearTimeout(timer)
  }, [autoFocus, internalValue, debounceMs, onChange])

  const handleClear = () => {
    setInternalValue('')
    onChange('')
  }
  return (
    <SearchInputView
      className={className}
      inputRef={inputRef}
      value={internalValue}
      onChange={(e) => setInternalValue(e.target.value)}
      placeholder={placeholder}
      onClear={handleClear}
    />
  )
}

function SearchInputView({
  className,
  inputRef,
  value,
  onChange,
  placeholder,
  onClear,
}: {
  className?: string
  inputRef: React.RefObject<HTMLInputElement | null>
  value: string
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
  placeholder: string
  onClear: () => void
}) {
  return (
    <div className={`relative ${className ?? ''}`}>
      <Search className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
      <Input
        ref={inputRef}
        type="search"
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        className="pr-9 pl-9"
      />
      {value ? (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          onClick={onClear}
          className="absolute top-1/2 right-1 size-7 -translate-y-1/2"
        >
          <X className="size-4" />
        </Button>
      ) : null}
    </div>
  )
}

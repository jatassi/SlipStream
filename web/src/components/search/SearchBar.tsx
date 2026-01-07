import { useState, useEffect, useRef } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Search, Loader2 } from 'lucide-react'
import { Input } from '@/components/ui/input'

export function SearchBar() {
  const navigate = useNavigate()
  const [searchQuery, setSearchQuery] = useState('')
  const [isSearching, setIsSearching] = useState(false)
  const debounceTimeoutRef = useRef<NodeJS.Timeout>()

  useEffect(() => {
    if (debounceTimeoutRef.current) {
      clearTimeout(debounceTimeoutRef.current)
    }

    if (searchQuery.trim()) {
      setIsSearching(true)
      debounceTimeoutRef.current = setTimeout(() => {
        navigate({ to: '/search', search: { q: searchQuery.trim() } })
        setIsSearching(false)
      }, 500)
    } else {
      setIsSearching(false)
    }

    return () => {
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current)
      }
    }
  }, [searchQuery, navigate])

  return (
    <div className="relative">
      {isSearching ? (
        <Loader2 className="absolute left-3 top-1/2 size-4 -translate-y-1/2 animate-spin text-foreground" />
      ) : (
        <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
      )}
      <Input
        type="search"
        placeholder="Search for and add movies, series..."
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
        className="pl-9"
      />
    </div>
  )
}
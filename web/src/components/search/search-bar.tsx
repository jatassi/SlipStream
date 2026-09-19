import { useEffect, useRef, useState } from 'react'

import { useNavigate } from '@tanstack/react-router'

import { SearchField } from '@/components/search/search-field'

function clearRef(ref: React.RefObject<ReturnType<typeof setTimeout> | null>) {
  if (ref.current) {
    clearTimeout(ref.current)
  }
}

export function SearchBar() {
  const navigate = useNavigate()
  const [searchQuery, setSearchQuery] = useState('')
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const commit = (value: string) => {
    void navigate({ to: '/search', search: { q: value } })
  }

  const handleSearchChange = (value: string) => {
    setSearchQuery(value)
    clearRef(timerRef)
    if (!value.trim()) {
      return
    }
    timerRef.current = setTimeout(() => {
      commit(value.trim())
    }, 500)
  }

  useEffect(() => () => clearRef(timerRef), [])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && searchQuery.trim()) {
      clearRef(timerRef)
      commit(searchQuery.trim())
    }
  }

  return (
    <SearchField
      value={searchQuery}
      onChange={handleSearchChange}
      onKeyDown={handleKeyDown}
      placeholder="Search..."
    />
  )
}

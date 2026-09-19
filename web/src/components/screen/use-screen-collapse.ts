import type { UIEvent } from 'react'
import { useState } from 'react'

export function useScreenCollapse(threshold: number) {
  const [collapsed, setCollapsed] = useState(false)

  const onScroll = (e: UIEvent<HTMLDivElement>) => {
    setCollapsed(e.currentTarget.scrollTop > threshold)
  }

  return { collapsed, onScroll }
}

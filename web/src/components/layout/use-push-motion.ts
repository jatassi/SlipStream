import { useState } from 'react'

import { useViewport } from '@/hooks/use-viewport'

import { lastNavInput } from './nav-input'
import { tabFromPathname } from './use-tab-nav'

export function usePushMotion(pathname: string): boolean {
  const shell = useViewport()
  const tab = tabFromPathname(pathname)
  const [prevTab, setPrevTab] = useState(tab)
  const [ready, setReady] = useState(false)
  const tabSwitch = prevTab !== tab
  const input = lastNavInput()

  if (prevTab !== tab) {
    setPrevTab(tab)
  }
  if (!ready) {
    setReady(true)
    return false
  }
  if (shell !== 'phone') {
    return false
  }
  if (tabSwitch) {
    return false
  }
  return input === 'pointer'
}

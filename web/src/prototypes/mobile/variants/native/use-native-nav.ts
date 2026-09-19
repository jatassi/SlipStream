import { useCallback, useState } from 'react'

export type NativeTab = 'home' | 'library' | 'activity' | 'search' | 'more'

export type PushedSpec =
  | { kind: 'detail'; id: number }
  | { kind: 'settings'; sectionId: string }
  | { kind: 'system' }

export type PushedScreen = PushedSpec & { key: string }

export type NativeNav = {
  tab: NativeTab
  setTab: (tab: NativeTab) => void
  stack: PushedScreen[]
  exitingKey: string | null
  push: (spec: PushedSpec) => void
  pop: () => void
  finishPop: () => void
}

let counter = 0

export function useNativeNav(): NativeNav {
  const [tab, setTab] = useState<NativeTab>('home')
  const [stack, setStack] = useState<PushedScreen[]>([])
  const [exitingKey, setExitingKey] = useState<string | null>(null)

  const push = useCallback((spec: PushedSpec) => {
    counter += 1
    setStack((prev) => [...prev, { ...spec, key: `s${counter}` }])
  }, [])

  const pop = useCallback(() => {
    setStack((prev) => {
      const top = prev.at(-1)
      if (top) {
        setExitingKey(top.key)
      }
      return prev
    })
  }, [])

  const finishPop = useCallback(() => {
    setStack((prev) => prev.slice(0, -1))
    setExitingKey(null)
  }, [])

  return { tab, setTab, stack, exitingKey, push, pop, finishPop }
}

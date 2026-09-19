import { useCallback } from 'react'

import { useRouter, useRouterState } from '@tanstack/react-router'

import { backLabelForPathname } from './push-routes'

export function usePushBack(): { label: string; onClick: () => void } | undefined {
  const router = useRouter()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const label = backLabelForPathname(pathname)
  const onClick = useCallback(() => {
    router.history.back()
  }, [router])

  if (label === undefined) {
    return undefined
  }
  return { label, onClick }
}

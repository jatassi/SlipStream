import { useEffect, useState } from 'react'

import { useNavigate } from '@tanstack/react-router'

import { useRestart } from '@/hooks'
import { usePortalAuthStore } from '@/stores'

export type SessionActions = ReturnType<typeof useSessionActions>

export function useSessionActions() {
  const navigate = useNavigate()
  const { logout } = usePortalAuthStore()
  const [restartOpen, setRestartOpen] = useState(false)
  const [logoutOpen, setLogoutOpen] = useState(false)
  const [countdown, setCountdown] = useState<number | null>(null)
  const restartMutation = useRestart()

  useEffect(() => {
    if (countdown === null) {
      return
    }
    if (countdown === 0) {
      globalThis.location.reload()
      return
    }
    const timer = setTimeout(() => setCountdown(countdown - 1), 1000)
    return () => clearTimeout(timer)
  }, [countdown])

  const handleAction = (action: string) => {
    if (action === 'restart') {
      setRestartOpen(true)
    } else if (action === 'logout') {
      setLogoutOpen(true)
    }
  }

  const handleRestart = async () => {
    await restartMutation.mutateAsync()
    setCountdown(5)
  }

  const handleLogout = () => {
    logout()
    void navigate({ to: '/requests/auth/login' })
  }

  return {
    restartOpen,
    setRestartOpen,
    logoutOpen,
    setLogoutOpen,
    countdown,
    isRestartPending: restartMutation.isPending,
    handleAction,
    handleRestart,
    handleLogout,
  }
}

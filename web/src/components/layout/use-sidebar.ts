import { useEffect, useState } from 'react'

import { useNavigate } from '@tanstack/react-router'

import { useRestart } from '@/hooks'
import { usePortalAuthStore, useUIStore } from '@/stores'

export function useSidebarActions() {
  const navigate = useNavigate()
  const { sidebarCollapsed, toggleSidebar } = useUIStore()
  const { logout } = usePortalAuthStore()
  const [showRestartDialog, setShowRestartDialog] = useState(false)
  const [showLogoutDialog, setShowLogoutDialog] = useState(false)
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
      setShowRestartDialog(true)
    } else if (action === 'logout') {
      setShowLogoutDialog(true)
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
    sidebarCollapsed,
    toggleSidebar,
    showRestartDialog,
    setShowRestartDialog,
    showLogoutDialog,
    setShowLogoutDialog,
    countdown,
    restartMutation,
    handleAction,
    handleRestart,
    handleLogout,
  }
}

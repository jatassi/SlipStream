import { useNavigate } from '@tanstack/react-router'
import { ArrowLeft, Loader2, LogOut } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { usePortalLogout } from '@/hooks'

const goBack = () => {
  globalThis.history.back()
}

export function SettingsHeader() {
  const navigate = useNavigate()
  const logoutMutation = usePortalLogout()

  const handleLogout = () => {
    logoutMutation.mutate(undefined, {
      onSuccess: () => {
        void navigate({ to: '/requests/auth/login' })
      },
    })
  }

  return (
    <div className="flex items-center gap-4">
      <Button variant="ghost" onClick={goBack} className="text-xs md:text-sm">
        <ArrowLeft className="mr-0.5 size-3 md:mr-1 md:size-4" />
        Back
      </Button>
      <h1 className="flex-1 text-xl font-bold md:text-2xl">Settings</h1>
      <Button
        variant="destructive"
        onClick={handleLogout}
        disabled={logoutMutation.isPending}
        className="text-xs md:text-sm"
      >
        {logoutMutation.isPending ? (
          <Loader2 className="mr-1 size-3 animate-spin md:mr-2 md:size-4" />
        ) : (
          <LogOut className="mr-1 size-3 md:mr-2 md:size-4" />
        )}
        Log Out
      </Button>
    </div>
  )
}

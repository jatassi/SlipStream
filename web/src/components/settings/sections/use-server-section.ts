import { useState } from 'react'

import { toast } from 'sonner'

import { useCheckFirewall, useFirewallStatus, useSettings, useStatus } from '@/hooks'

export function useServerSection() {
  const { data: settings, isLoading, isError, refetch } = useSettings()
  const { data: status } = useStatus()
  const { data: firewallStatus, isLoading: firewallLoading } = useFirewallStatus()
  const checkFirewallMutation = useCheckFirewall()
  const [isCopied, setIsCopied] = useState(false)

  const portConflict = !!(status?.configuredPort && status.actualPort !== status.configuredPort)

  const handleCopyLogPath = async () => {
    if (!settings) {
      return
    }
    try {
      await navigator.clipboard.writeText(settings.logPath)
      setIsCopied(true)
      toast.success('Log path copied to clipboard')
      setTimeout(() => setIsCopied(false), 2000)
    } catch {
      toast.error('Failed to copy to clipboard')
    }
  }

  const handleCheckFirewall = () => {
    checkFirewallMutation.mutate()
  }

  return {
    settings,
    isLoading,
    isError,
    refetch,
    status,
    firewallStatus,
    firewallLoading,
    isCheckingFirewall: checkFirewallMutation.isPending,
    isCopied,
    portConflict,
    handleCopyLogPath,
    handleCheckFirewall,
  }
}

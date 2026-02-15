import { AlertCircle, CheckCircle2, Loader2, RefreshCw, Shield, XCircle } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import type { FirewallStatus } from '@/types'

type FirewallStatusPanelProps = {
  firewallStatus: FirewallStatus | undefined
  firewallLoading: boolean
  isChecking: boolean
  onCheck: () => void
}

function PortStatusIndicator({ firewallStatus }: { firewallStatus: FirewallStatus }) {
  if (firewallStatus.isListening && firewallStatus.firewallAllows) {
    return (
      <>
        <CheckCircle2 className="size-4 text-green-500" />
        <span className="text-sm text-green-600 dark:text-green-400">
          Port {firewallStatus.port} is open for external access
        </span>
      </>
    )
  }

  if (firewallStatus.isListening && firewallStatus.firewallEnabled) {
    return (
      <>
        <XCircle className="size-4 text-red-500" />
        <span className="text-sm text-red-600 dark:text-red-400">
          Port {firewallStatus.port} may be blocked by firewall
        </span>
      </>
    )
  }

  if (firewallStatus.isListening) {
    return (
      <>
        <AlertCircle className="size-4 text-amber-500" />
        <span className="text-sm text-amber-600 dark:text-amber-400">
          No firewall detected - port should be accessible
        </span>
      </>
    )
  }

  return (
    <>
      <XCircle className="size-4 text-red-500" />
      <span className="text-sm text-red-600 dark:text-red-400">
        Port {firewallStatus.port} is not listening
      </span>
    </>
  )
}

function FirewallDetails({ firewallStatus }: { firewallStatus: FirewallStatus }) {
  return (
    <div className="space-y-2 rounded-lg border p-3">
      <div className="flex items-center gap-2">
        <Shield className="text-muted-foreground size-4" />
        <span className="text-sm font-medium">
          {firewallStatus.firewallName ?? 'Firewall'}
        </span>
        <Badge
          variant={firewallStatus.firewallEnabled ? 'secondary' : 'outline'}
          className="text-xs"
        >
          {firewallStatus.firewallEnabled ? 'Enabled' : 'Disabled'}
        </Badge>
      </div>
      <div className="flex items-center gap-2">
        <PortStatusIndicator firewallStatus={firewallStatus} />
      </div>
      {firewallStatus.message &&
      firewallStatus.firewallEnabled &&
      !firewallStatus.firewallAllows ? (
        <p className="text-muted-foreground text-xs">
          You may need to add a firewall rule to allow incoming connections on port{' '}
          {firewallStatus.port}.
        </p>
      ) : null}
    </div>
  )
}

export function FirewallStatusPanel({
  firewallStatus,
  firewallLoading,
  isChecking,
  onCheck,
}: FirewallStatusPanelProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <Label>Firewall Status</Label>
        <Button variant="ghost" size="sm" onClick={onCheck} disabled={isChecking}>
          {isChecking ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <RefreshCw className="size-4" />
          )}
          <span className="ml-1">Check</span>
        </Button>
      </div>
      {firewallLoading ? (
        <div className="text-muted-foreground flex items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" />
          Checking firewall status...
        </div>
      ) : null}
      {!firewallLoading && firewallStatus ? (
        <FirewallDetails firewallStatus={firewallStatus} />
      ) : null}
    </div>
  )
}

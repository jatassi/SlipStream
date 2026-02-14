import { useEffect, useState } from 'react'

import {
  AlertCircle,
  Check,
  CheckCircle2,
  Copy,
  Loader2,
  RefreshCw,
  Shield,
  XCircle,
} from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/ErrorState'
import { LoadingState } from '@/components/data/LoadingState'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from '@/components/ui/input-group'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { useCheckFirewall, useFirewallStatus, useSettings, useStatus } from '@/hooks'

const LOG_LEVELS = [
  { value: 'trace', label: 'Trace' },
  { value: 'debug', label: 'Debug' },
  { value: 'info', label: 'Info' },
  { value: 'warn', label: 'Warn' },
  { value: 'error', label: 'Error' },
]

type LogRotationSettings = {
  maxSizeMB: number
  maxBackups: number
  maxAgeDays: number
  compress: boolean
}

type ServerSectionProps = {
  port: string
  onPortChange: (port: string) => void
  logLevel: string
  onLogLevelChange: (level: string) => void
  logRotation: LogRotationSettings
  onLogRotationChange: (settings: LogRotationSettings) => void
  externalAccessEnabled: boolean
  onExternalAccessChange: (enabled: boolean) => void
}

export function ServerSection({
  port,
  onPortChange,
  logLevel,
  onLogLevelChange,
  logRotation,
  onLogRotationChange,
  externalAccessEnabled,
  onExternalAccessChange,
}: ServerSectionProps) {
  const { data: settings, isLoading, isError, refetch } = useSettings()
  const { data: status } = useStatus()
  const { data: firewallStatus, isLoading: firewallLoading } = useFirewallStatus()
  const checkFirewallMutation = useCheckFirewall()
  const [isCopied, setIsCopied] = useState(false)

  const portConflict = status?.configuredPort && status.actualPort !== status.configuredPort

  useEffect(() => {
    if (settings) {
      onPortChange(settings.serverPort.toString())
      onLogLevelChange(settings.logLevel)
      onLogRotationChange({
        maxSizeMB: settings.logMaxSizeMB,
        maxBackups: settings.logMaxBackups,
        maxAgeDays: settings.logMaxAgeDays,
        compress: settings.logCompress,
      })
      onExternalAccessChange(settings.externalAccessEnabled)
    }
  }, [settings, onPortChange, onLogLevelChange, onLogRotationChange, onExternalAccessChange])

  const handleCopyLogPath = async () => {
    if (settings) {
      try {
        await navigator.clipboard.writeText(settings.logPath)
        setIsCopied(true)
        toast.success('Log path copied to clipboard')
        setTimeout(() => setIsCopied(false), 2000)
      } catch {
        toast.error('Failed to copy to clipboard')
      }
    }
  }

  if (isLoading) {
    return <LoadingState variant="list" count={3} />
  }

  if (isError) {
    return <ErrorState onRetry={refetch} />
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="port">Port</Label>
        <Input
          id="port"
          type="number"
          value={port}
          onChange={(e) => onPortChange(e.target.value)}
          placeholder="8080"
        />
        {portConflict ? (
          <p className="text-sm text-amber-600 dark:text-amber-500">
            Port {status.configuredPort} was in use. Server is running on port {status.actualPort}.
            Restart required to apply port changes.
          </p>
        ) : null}
      </div>

      <div className="space-y-2">
        <Label htmlFor="externalAccess">External Access</Label>
        <div className="flex items-center gap-2">
          <Switch
            id="externalAccess"
            checked={externalAccessEnabled}
            onCheckedChange={onExternalAccessChange}
          />
          <span className="text-muted-foreground text-sm">
            {externalAccessEnabled ? 'Enabled' : 'Disabled'}
          </span>
        </div>
        <p className="text-muted-foreground text-sm">
          {externalAccessEnabled
            ? 'Server is accessible from other devices on your network'
            : 'Server is only accessible from this machine (localhost)'}
        </p>
        {externalAccessEnabled ? (
          <p className="text-sm text-amber-600 dark:text-amber-500">
            Warning: Enabling external access exposes your server to other devices on your network.
            Ensure you have proper authentication enabled.
          </p>
        ) : null}
      </div>

      {externalAccessEnabled ? (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <Label>Firewall Status</Label>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => checkFirewallMutation.mutate()}
              disabled={checkFirewallMutation.isPending}
            >
              {checkFirewallMutation.isPending ? (
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
          ) : firewallStatus ? (
            <div className="space-y-2 rounded-lg border p-3">
              <div className="flex items-center gap-2">
                <Shield className="text-muted-foreground size-4" />
                <span className="text-sm font-medium">
                  {firewallStatus.firewallName || 'Firewall'}
                </span>
                {firewallStatus.firewallEnabled ? (
                  <Badge variant="secondary" className="text-xs">
                    Enabled
                  </Badge>
                ) : (
                  <Badge variant="outline" className="text-xs">
                    Disabled
                  </Badge>
                )}
              </div>
              <div className="flex items-center gap-2">
                {firewallStatus.isListening && firewallStatus.firewallAllows ? (
                  <>
                    <CheckCircle2 className="size-4 text-green-500" />
                    <span className="text-sm text-green-600 dark:text-green-400">
                      Port {firewallStatus.port} is open for external access
                    </span>
                  </>
                ) : firewallStatus.isListening ? (
                  firewallStatus.firewallEnabled ? (
                    <>
                      <XCircle className="size-4 text-red-500" />
                      <span className="text-sm text-red-600 dark:text-red-400">
                        Port {firewallStatus.port} may be blocked by firewall
                      </span>
                    </>
                  ) : (
                    <>
                      <AlertCircle className="size-4 text-amber-500" />
                      <span className="text-sm text-amber-600 dark:text-amber-400">
                        No firewall detected - port should be accessible
                      </span>
                    </>
                  )
                ) : (
                  <>
                    <XCircle className="size-4 text-red-500" />
                    <span className="text-sm text-red-600 dark:text-red-400">
                      Port {firewallStatus.port} is not listening
                    </span>
                  </>
                )}
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
          ) : null}
        </div>
      ) : null}

      <div className="space-y-2">
        <Label htmlFor="logLevel">Log Level</Label>
        <Select value={logLevel} onValueChange={(v) => v && onLogLevelChange(v)}>
          <SelectTrigger>
            {LOG_LEVELS.find((l) => l.value === logLevel)?.label || 'Info'}
          </SelectTrigger>
          <SelectContent>
            {LOG_LEVELS.map((level) => (
              <SelectItem key={level.value} value={level.value}>
                {level.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label>Log Files</Label>
        <InputGroup>
          <InputGroupInput value={settings?.logPath || ''} readOnly className="font-mono text-sm" />
          <InputGroupAddon align="inline-end">
            <InputGroupButton
              aria-label="Copy"
              title="Copy path"
              size="icon-xs"
              onClick={handleCopyLogPath}
            >
              {isCopied ? <Check className="size-4" /> : <Copy className="size-4" />}
            </InputGroupButton>
          </InputGroupAddon>
        </InputGroup>
        <p className="text-muted-foreground text-sm">Location where log files are stored</p>
      </div>

      <div className="space-y-4 border-t pt-4">
        <div>
          <h4 className="mb-1 text-sm font-medium">Log Rotation</h4>
          <p className="text-muted-foreground text-sm">
            Configure automatic log file rotation to manage disk space
          </p>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="maxSizeMB">Max File Size (MB)</Label>
            <Input
              id="maxSizeMB"
              type="number"
              min={1}
              max={100}
              value={logRotation.maxSizeMB}
              onChange={(e) =>
                onLogRotationChange({
                  ...logRotation,
                  maxSizeMB: Number.parseInt(e.target.value) || 10,
                })
              }
            />
            <p className="text-muted-foreground text-xs">Rotate when file exceeds this size</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="maxBackups">Max Backup Files</Label>
            <Input
              id="maxBackups"
              type="number"
              min={1}
              max={20}
              value={logRotation.maxBackups}
              onChange={(e) =>
                onLogRotationChange({
                  ...logRotation,
                  maxBackups: Number.parseInt(e.target.value) || 5,
                })
              }
            />
            <p className="text-muted-foreground text-xs">Number of old files to keep</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="maxAgeDays">Max Age (Days)</Label>
            <Input
              id="maxAgeDays"
              type="number"
              min={1}
              max={365}
              value={logRotation.maxAgeDays}
              onChange={(e) =>
                onLogRotationChange({
                  ...logRotation,
                  maxAgeDays: Number.parseInt(e.target.value) || 30,
                })
              }
            />
            <p className="text-muted-foreground text-xs">Delete files older than this</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="compress">Compress Old Logs</Label>
            <div className="flex items-center gap-2 pt-1">
              <Switch
                id="compress"
                checked={logRotation.compress}
                onCheckedChange={(checked) =>
                  onLogRotationChange({ ...logRotation, compress: checked })
                }
              />
              <span className="text-muted-foreground text-sm">
                {logRotation.compress ? 'Enabled' : 'Disabled'}
              </span>
            </div>
            <p className="text-muted-foreground text-xs">Gzip rotated log files</p>
          </div>
        </div>
      </div>
    </div>
  )
}

import { useState, useEffect } from 'react'
import { Copy, Check } from 'lucide-react'
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
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { useSettings, useStatus } from '@/hooks'
import { toast } from 'sonner'

const LOG_LEVELS = [
  { value: 'trace', label: 'Trace' },
  { value: 'debug', label: 'Debug' },
  { value: 'info', label: 'Info' },
  { value: 'warn', label: 'Warn' },
  { value: 'error', label: 'Error' },
]

interface LogRotationSettings {
  maxSizeMB: number
  maxBackups: number
  maxAgeDays: number
  compress: boolean
}

interface ServerSectionProps {
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
  const [isCopied, setIsCopied] = useState(false)

  const portConflict = status?.configuredPort && status.actualPort !== status.configuredPort

  useEffect(() => {
    if (settings) {
      onPortChange(settings.serverPort?.toString() || '8080')
      onLogLevelChange(settings.logLevel || 'info')
      onLogRotationChange({
        maxSizeMB: settings.logMaxSizeMB ?? 10,
        maxBackups: settings.logMaxBackups ?? 5,
        maxAgeDays: settings.logMaxAgeDays ?? 30,
        compress: settings.logCompress ?? false,
      })
      onExternalAccessChange(settings.externalAccessEnabled ?? false)
    }
  }, [settings, onPortChange, onLogLevelChange, onLogRotationChange, onExternalAccessChange])

  const handleCopyLogPath = () => {
    if (settings?.logPath) {
      navigator.clipboard.writeText(settings.logPath)
      setIsCopied(true)
      toast.success('Log path copied to clipboard')
      setTimeout(() => setIsCopied(false), 2000)
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
        {portConflict && (
          <p className="text-sm text-amber-600 dark:text-amber-500">
            Port {status.configuredPort} was in use. Server is running on port {status.actualPort}.
            Restart required to apply port changes.
          </p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="externalAccess">External Access</Label>
        <div className="flex items-center gap-2">
          <Switch
            id="externalAccess"
            checked={externalAccessEnabled}
            onCheckedChange={onExternalAccessChange}
          />
          <span className="text-sm text-muted-foreground">
            {externalAccessEnabled ? 'Enabled' : 'Disabled'}
          </span>
        </div>
        <p className="text-sm text-muted-foreground">
          {externalAccessEnabled
            ? 'Server is accessible from other devices on your network'
            : 'Server is only accessible from this machine (localhost)'}
        </p>
        {externalAccessEnabled && (
          <p className="text-sm text-amber-600 dark:text-amber-500">
            Warning: Enabling external access exposes your server to other devices on your network.
            Ensure you have proper authentication enabled.
          </p>
        )}
      </div>

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
          <InputGroupInput
            value={settings?.logPath || ''}
            readOnly
            className="font-mono text-sm"
          />
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
        <p className="text-sm text-muted-foreground">
          Location where log files are stored
        </p>
      </div>

      <div className="pt-4 border-t space-y-4">
        <div>
          <h4 className="text-sm font-medium mb-1">Log Rotation</h4>
          <p className="text-sm text-muted-foreground">
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
                onLogRotationChange({ ...logRotation, maxSizeMB: parseInt(e.target.value) || 10 })
              }
            />
            <p className="text-xs text-muted-foreground">Rotate when file exceeds this size</p>
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
                onLogRotationChange({ ...logRotation, maxBackups: parseInt(e.target.value) || 5 })
              }
            />
            <p className="text-xs text-muted-foreground">Number of old files to keep</p>
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
                onLogRotationChange({ ...logRotation, maxAgeDays: parseInt(e.target.value) || 30 })
              }
            />
            <p className="text-xs text-muted-foreground">Delete files older than this</p>
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
              <span className="text-sm text-muted-foreground">
                {logRotation.compress ? 'Enabled' : 'Disabled'}
              </span>
            </div>
            <p className="text-xs text-muted-foreground">Gzip rotated log files</p>
          </div>
        </div>
      </div>
    </div>
  )
}

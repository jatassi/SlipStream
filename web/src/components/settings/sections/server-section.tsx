import { useEffect } from 'react'

import { Check, Copy } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
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

import { FirewallStatusPanel } from './firewall-status-panel'
import { useServerSection } from './use-server-section'

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

function PortField({
  port,
  onChange,
  portConflict,
  configuredPort,
  actualPort,
}: {
  port: string
  onChange: (v: string) => void
  portConflict: boolean
  configuredPort?: number
  actualPort?: number
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor="port">Port</Label>
      <Input id="port" type="number" value={port} onChange={(e) => onChange(e.target.value)} placeholder="8080" />
      {portConflict ? (
        <p className="text-sm text-amber-600 dark:text-amber-500">
          Port {configuredPort} was in use. Server is running on port {actualPort}. Restart required to apply port changes.
        </p>
      ) : null}
    </div>
  )
}

function ExternalAccessField({
  enabled,
  onChange,
}: {
  enabled: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor="externalAccess">External Access</Label>
      <div className="flex items-center gap-2">
        <Switch id="externalAccess" checked={enabled} onCheckedChange={onChange} />
        <span className="text-muted-foreground text-sm">{enabled ? 'Enabled' : 'Disabled'}</span>
      </div>
      <p className="text-muted-foreground text-sm">
        {enabled ? 'Server is accessible from other devices on your network' : 'Server is only accessible from this machine (localhost)'}
      </p>
      {enabled ? (
        <p className="text-sm text-amber-600 dark:text-amber-500">
          Warning: Enabling external access exposes your server to other devices on your network. Ensure you have proper authentication enabled.
        </p>
      ) : null}
    </div>
  )
}

function LogLevelField({ logLevel, onChange }: { logLevel: string; onChange: (v: string) => void }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="logLevel">Log Level</Label>
      <Select value={logLevel} onValueChange={(v) => v && onChange(v)}>
        <SelectTrigger>{LOG_LEVELS.find((l) => l.value === logLevel)?.label ?? 'Info'}</SelectTrigger>
        <SelectContent>
          {LOG_LEVELS.map((level) => (
            <SelectItem key={level.value} value={level.value}>{level.label}</SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function LogPathField({ logPath, isCopied, onCopy }: { logPath: string; isCopied: boolean; onCopy: () => void }) {
  return (
    <div className="space-y-2">
      <Label>Log Files</Label>
      <InputGroup>
        <InputGroupInput value={logPath} readOnly className="font-mono text-sm" />
        <InputGroupAddon align="inline-end">
          <InputGroupButton aria-label="Copy" title="Copy path" size="icon-xs" onClick={onCopy}>
            {isCopied ? <Check className="size-4" /> : <Copy className="size-4" />}
          </InputGroupButton>
        </InputGroupAddon>
      </InputGroup>
      <p className="text-muted-foreground text-sm">Location where log files are stored</p>
    </div>
  )
}

function LogRotationField({ logRotation, onChange }: { logRotation: LogRotationSettings; onChange: (s: LogRotationSettings) => void }) {
  const updateField = (field: keyof LogRotationSettings, fallback: number) => (e: React.ChangeEvent<HTMLInputElement>) =>
    onChange({ ...logRotation, [field]: Number.parseInt(e.target.value) || fallback })

  return (
    <div className="space-y-4 border-t pt-4">
      <div>
        <h4 className="mb-1 text-sm font-medium">Log Rotation</h4>
        <p className="text-muted-foreground text-sm">Configure automatic log file rotation to manage disk space</p>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="maxSizeMB">Max File Size (MB)</Label>
          <Input id="maxSizeMB" type="number" min={1} max={100} value={logRotation.maxSizeMB} onChange={updateField('maxSizeMB', 10)} />
          <p className="text-muted-foreground text-xs">Rotate when file exceeds this size</p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="maxBackups">Max Backup Files</Label>
          <Input id="maxBackups" type="number" min={1} max={20} value={logRotation.maxBackups} onChange={updateField('maxBackups', 5)} />
          <p className="text-muted-foreground text-xs">Number of old files to keep</p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="maxAgeDays">Max Age (Days)</Label>
          <Input id="maxAgeDays" type="number" min={1} max={365} value={logRotation.maxAgeDays} onChange={updateField('maxAgeDays', 30)} />
          <p className="text-muted-foreground text-xs">Delete files older than this</p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="compress">Compress Old Logs</Label>
          <div className="flex items-center gap-2 pt-1">
            <Switch id="compress" checked={logRotation.compress} onCheckedChange={(checked) => onChange({ ...logRotation, compress: checked })} />
            <span className="text-muted-foreground text-sm">{logRotation.compress ? 'Enabled' : 'Disabled'}</span>
          </div>
          <p className="text-muted-foreground text-xs">Gzip rotated log files</p>
        </div>
      </div>
    </div>
  )
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
  const {
    settings, isLoading, isError, refetch, status,
    firewallStatus, firewallLoading, isCheckingFirewall,
    isCopied, portConflict, handleCopyLogPath, handleCheckFirewall,
  } = useServerSection()

  useEffect(() => {
    if (!settings) {return}
    onPortChange(settings.serverPort.toString())
    onLogLevelChange(settings.logLevel)
    onLogRotationChange({ maxSizeMB: settings.logMaxSizeMB, maxBackups: settings.logMaxBackups, maxAgeDays: settings.logMaxAgeDays, compress: settings.logCompress })
    onExternalAccessChange(settings.externalAccessEnabled)
  }, [settings, onPortChange, onLogLevelChange, onLogRotationChange, onExternalAccessChange])

  if (isLoading) {return <LoadingState variant="list" count={3} />}
  if (isError) {return <ErrorState onRetry={refetch} />}

  return (
    <div className="space-y-4">
      <PortField port={port} onChange={onPortChange} portConflict={portConflict} configuredPort={status?.configuredPort} actualPort={status?.actualPort} />
      <ExternalAccessField enabled={externalAccessEnabled} onChange={onExternalAccessChange} />
      {externalAccessEnabled ? (
        <FirewallStatusPanel firewallStatus={firewallStatus} firewallLoading={firewallLoading} isChecking={isCheckingFirewall} onCheck={handleCheckFirewall} />
      ) : null}
      <LogLevelField logLevel={logLevel} onChange={onLogLevelChange} />
      <LogPathField logPath={settings?.logPath ?? ''} isCopied={isCopied} onCopy={handleCopyLogPath} />
      <LogRotationField logRotation={logRotation} onChange={onLogRotationChange} />
    </div>
  )
}

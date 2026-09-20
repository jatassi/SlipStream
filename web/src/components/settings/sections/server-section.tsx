import { useEffect, useState } from 'react'

import { Check, Copy } from 'lucide-react'

import { Group, Row } from '@/components/grouped-list'
import { InputRow, SelectRow, SwitchRow } from '@/components/settings/control-row'
import { SectionError, SectionLoading } from '@/components/settings/section-state'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'

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

function portFooter(portConflict: boolean, configuredPort?: number, actualPort?: number) {
  if (portConflict) {
    return `Port ${configuredPort} was in use. Server is running on port ${actualPort}. Restart required to apply port changes.`
  }
  return 'The port the server listens on. Restart required to apply port changes.'
}

function ExternalAccessGroup({
  enabled,
  onChange,
  children,
}: {
  enabled: boolean
  onChange: (v: boolean) => void
  children?: React.ReactNode
}) {
  const [confirmOpen, setConfirmOpen] = useState(false)

  const handleToggle = (checked: boolean) => {
    if (checked) {
      setConfirmOpen(true)
      return
    }
    onChange(false)
  }

  return (
    <Group
      header="External Access"
      footer={
        enabled
          ? 'Server is accessible from other devices on your network.'
          : 'Server is only accessible from this machine (localhost).'
      }
    >
      <SwitchRow label="External Access" checked={enabled} onCheckedChange={handleToggle} />
      {children}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Enable External Access</AlertDialogTitle>
            <AlertDialogDescription>
              This will expose your server to other devices on your network. Ensure you have proper
              authentication enabled before proceeding.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={() => onChange(true)}>Enable</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Group>
  )
}

function LogRotationGroup({
  logRotation,
  onChange,
}: {
  logRotation: LogRotationSettings
  onChange: (s: LogRotationSettings) => void
}) {
  const updateField =
    (field: keyof LogRotationSettings, fallback: number) =>
    (e: React.ChangeEvent<HTMLInputElement>) =>
      onChange({ ...logRotation, [field]: Number.parseInt(e.target.value) || fallback })

  return (
    <Group
      header="Log Rotation"
      footer="Rotate when a file exceeds the maximum size, keep this many old files, delete files older than the maximum age, and optionally gzip rotated logs."
    >
      <InputRow
        label="Max File Size (MB)"
        type="number"
        min={1}
        max={100}
        value={logRotation.maxSizeMB}
        onChange={updateField('maxSizeMB', 10)}
      />
      <InputRow
        label="Max Backup Files"
        type="number"
        min={1}
        max={20}
        value={logRotation.maxBackups}
        onChange={updateField('maxBackups', 5)}
      />
      <InputRow
        label="Max Age (Days)"
        type="number"
        min={1}
        max={365}
        value={logRotation.maxAgeDays}
        onChange={updateField('maxAgeDays', 30)}
      />
      <SwitchRow
        label="Compress Old Logs"
        checked={logRotation.compress}
        onCheckedChange={(checked) => onChange({ ...logRotation, compress: checked })}
      />
    </Group>
  )
}

function useSyncServerSettings(props: ServerSectionProps, settings: ReturnType<typeof useServerSection>['settings']) {
  const { onPortChange, onLogLevelChange, onLogRotationChange, onExternalAccessChange } = props
  useEffect(() => {
    if (!settings) {
      return
    }
    onPortChange(settings.serverPort.toString())
    onLogLevelChange(settings.logLevel)
    onLogRotationChange({
      maxSizeMB: settings.logMaxSizeMB,
      maxBackups: settings.logMaxBackups,
      maxAgeDays: settings.logMaxAgeDays,
      compress: settings.logCompress,
    })
    onExternalAccessChange(settings.externalAccessEnabled)
  }, [settings, onPortChange, onLogLevelChange, onLogRotationChange, onExternalAccessChange])
}

function LoggingGroup({
  logLevel,
  onLogLevelChange,
  logPath,
  isCopied,
  onCopy,
}: {
  logLevel: string
  onLogLevelChange: (level: string) => void
  logPath: string
  isCopied: boolean
  onCopy: () => void
}) {
  return (
    <Group header="Logging" footer={`Log files are stored at ${logPath}`}>
      <SelectRow
        label="Log Level"
        value={logLevel}
        onChange={onLogLevelChange}
        options={LOG_LEVELS}
      />
      <Row
        title="Copy Log Path"
        onClick={onCopy}
        trailing={isCopied ? <Check className="size-4" /> : <Copy className="size-4" />}
      />
    </Group>
  )
}

function NetworkGroup({
  port,
  onPortChange,
  portConflict,
  configuredPort,
  actualPort,
}: {
  port: string
  onPortChange: (port: string) => void
  portConflict: boolean
  configuredPort?: number
  actualPort?: number
}) {
  return (
    <Group header="Network" footer={portFooter(portConflict, configuredPort, actualPort)}>
      <InputRow
        label="Port"
        type="number"
        value={port}
        placeholder="8080"
        onChange={(e) => onPortChange(e.target.value)}
      />
    </Group>
  )
}

export function ServerSection(props: ServerSectionProps) {
  const server = useServerSection()
  useSyncServerSettings(props, server.settings)

  if (server.isLoading) {
    return <SectionLoading count={3} />
  }
  if (server.isError) {
    return <SectionError onRetry={server.refetch} />
  }

  return (
    <>
      <NetworkGroup
        port={props.port}
        onPortChange={props.onPortChange}
        portConflict={server.portConflict}
        configuredPort={server.status?.configuredPort}
        actualPort={server.status?.actualPort}
      />
      <ExternalAccessGroup
        enabled={props.externalAccessEnabled}
        onChange={props.onExternalAccessChange}
      >
        {props.externalAccessEnabled ? (
          <div className="px-4 py-3">
            <FirewallStatusPanel
              firewallStatus={server.firewallStatus}
              firewallLoading={server.firewallLoading}
              isChecking={server.isCheckingFirewall}
              onCheck={server.handleCheckFirewall}
            />
          </div>
        ) : undefined}
      </ExternalAccessGroup>
      <LoggingGroup
        logLevel={props.logLevel}
        onLogLevelChange={props.onLogLevelChange}
        logPath={server.settings?.logPath ?? ''}
        isCopied={server.isCopied}
        onCopy={() => void server.handleCopyLogPath()}
      />
      <LogRotationGroup logRotation={props.logRotation} onChange={props.onLogRotationChange} />
    </>
  )
}

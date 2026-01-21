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
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { useSettings } from '@/hooks'
import { toast } from 'sonner'

const LOG_LEVELS = [
  { value: 'trace', label: 'Trace' },
  { value: 'debug', label: 'Debug' },
  { value: 'info', label: 'Info' },
  { value: 'warn', label: 'Warn' },
  { value: 'error', label: 'Error' },
]

interface ServerSectionProps {
  port: string
  onPortChange: (port: string) => void
  logLevel: string
  onLogLevelChange: (level: string) => void
}

export function ServerSection({
  port,
  onPortChange,
  logLevel,
  onLogLevelChange,
}: ServerSectionProps) {
  const { data: settings, isLoading, isError, refetch } = useSettings()
  const [isCopied, setIsCopied] = useState(false)

  useEffect(() => {
    if (settings) {
      onPortChange(settings.serverPort?.toString() || '8080')
      onLogLevelChange(settings.logLevel || 'info')
    }
  }, [settings, onPortChange, onLogLevelChange])

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
    </div>
  )
}

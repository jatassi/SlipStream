import { useState, useEffect } from 'react'
import { Save, Lock, Copy, Check } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
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
import { PasskeyManager, ChangePinDialog } from '@/components/portal'
import { useSettings, useUpdateSettings } from '@/hooks'
import { toast } from 'sonner'

export function GeneralSettingsPage() {
  const { data: settings, isLoading, isError, refetch } = useSettings()
  const updateMutation = useUpdateSettings()

  const [port, setPort] = useState('')
  const [logLevel, setLogLevel] = useState('')
  const [pinDialogOpen, setPinDialogOpen] = useState(false)
  const [isCopied, setIsCopied] = useState(false)

  useEffect(() => {
    if (settings) {
      setPort(settings.serverPort?.toString() || '8080')
      setLogLevel(settings.logLevel || 'info')
    }
  }, [settings])

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync({
        serverPort: parseInt(port),
        logLevel,
      })
      toast.success('Settings saved')
    } catch {
      toast.error('Failed to save settings')
    }
  }

  const handleCopyLogPath = () => {
    if (settings?.logPath) {
      navigator.clipboard.writeText(settings.logPath)
      setIsCopied(true)
      toast.success('Log path copied to clipboard')
      setTimeout(() => setIsCopied(false), 2000)
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="General Settings" />
        <LoadingState variant="list" count={3} />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="General Settings" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="General Settings"
        description="Application configuration"
        breadcrumbs={[
          { label: 'Settings', href: '/settings' },
          { label: 'General' },
        ]}
        actions={
          <Button onClick={handleSave} disabled={updateMutation.isPending}>
            <Save className="size-4 mr-2" />
            Save Changes
          </Button>
        }
      />

      <div className="space-y-6 max-w-2xl">
        {/* Server settings */}
        <Card>
          <CardHeader>
            <CardTitle>Server</CardTitle>
            <CardDescription>
              Server configuration options
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="port">Port</Label>
              <Input
                id="port"
                type="number"
                value={port}
                onChange={(e) => setPort(e.target.value)}
                placeholder="8080"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="logLevel">Log Level</Label>
              <Select value={logLevel} onValueChange={(v) => v && setLogLevel(v)}>
                <SelectTrigger>
                  {{ trace: 'Trace', debug: 'Debug', info: 'Info', warn: 'Warn', error: 'Error' }[logLevel] || 'Info'}
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="trace">Trace</SelectItem>
                  <SelectItem value="debug">Debug</SelectItem>
                  <SelectItem value="info">Info</SelectItem>
                  <SelectItem value="warn">Warn</SelectItem>
                  <SelectItem value="error">Error</SelectItem>
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
          </CardContent>
        </Card>

        {/* Authentication */}
        <Card>
          <CardHeader>
            <CardTitle>Authentication</CardTitle>
            <CardDescription>
              Secure access to the web interface
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div>
              <Label className="text-base">PIN</Label>
              <p className="text-sm text-muted-foreground mb-3">
                Update your account PIN
              </p>
              <Button onClick={() => setPinDialogOpen(true)}>
                <Lock className="size-4 mr-2" />
                Change PIN...
              </Button>
            </div>

            <div className="border-t pt-6">
              <PasskeyManager />
            </div>
          </CardContent>
        </Card>

        <ChangePinDialog open={pinDialogOpen} onOpenChange={setPinDialogOpen} />
      </div>
    </div>
  )
}

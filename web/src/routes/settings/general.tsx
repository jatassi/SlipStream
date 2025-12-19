import { useState } from 'react'
import { Save, RefreshCw, Copy } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { useSettings, useUpdateSettings, useRegenerateApiKey } from '@/hooks'
import { toast } from 'sonner'

export function GeneralSettingsPage() {
  const { data: settings, isLoading, isError, refetch } = useSettings()
  const updateMutation = useUpdateSettings()
  const regenerateMutation = useRegenerateApiKey()

  const [port, setPort] = useState(settings?.serverPort?.toString() || '8080')
  const [logLevel, setLogLevel] = useState(settings?.logLevel || 'info')
  const [authEnabled, setAuthEnabled] = useState(settings?.authEnabled || false)
  const [password, setPassword] = useState('')

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync({
        serverPort: parseInt(port),
        logLevel,
        authEnabled,
        password: password || undefined,
      })
      toast.success('Settings saved')
      setPassword('')
    } catch {
      toast.error('Failed to save settings')
    }
  }

  const handleRegenerateApiKey = async () => {
    try {
      await regenerateMutation.mutateAsync()
      toast.success('API key regenerated')
    } catch {
      toast.error('Failed to regenerate API key')
    }
  }

  const handleCopyApiKey = () => {
    if (settings?.apiKey) {
      navigator.clipboard.writeText(settings.apiKey)
      toast.success('API key copied to clipboard')
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
                  <SelectValue />
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
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Enable Authentication</Label>
                <p className="text-sm text-muted-foreground">
                  Require a password to access the web interface
                </p>
              </div>
              <Switch
                checked={authEnabled}
                onCheckedChange={setAuthEnabled}
              />
            </div>

            {authEnabled && (
              <div className="space-y-2">
                <Label htmlFor="password">New Password</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Enter new password"
                />
                <p className="text-xs text-muted-foreground">
                  Leave blank to keep current password
                </p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* API Key */}
        <Card>
          <CardHeader>
            <CardTitle>API Key</CardTitle>
            <CardDescription>
              Used for external API access
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-2">
              <Input
                value={settings?.apiKey || ''}
                readOnly
                className="font-mono text-sm"
              />
              <Button variant="outline" size="icon" onClick={handleCopyApiKey}>
                <Copy className="size-4" />
              </Button>
            </div>

            <ConfirmDialog
              trigger={
                <Button variant="outline">
                  <RefreshCw className="size-4 mr-2" />
                  Regenerate API Key
                </Button>
              }
              title="Regenerate API Key"
              description="Are you sure you want to regenerate the API key? Any applications using the current key will need to be updated."
              confirmLabel="Regenerate"
              onConfirm={handleRegenerateApiKey}
            />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

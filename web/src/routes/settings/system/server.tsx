import { useState, useCallback } from 'react'
import { Save } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { ServerSection } from '@/components/settings'
import { SystemNav } from './SystemNav'
import { useUpdateSettings } from '@/hooks'
import { toast } from 'sonner'

export function ServerPage() {
  const updateMutation = useUpdateSettings()

  const [port, setPort] = useState('')
  const [logLevel, setLogLevel] = useState('')
  const [initialPort, setInitialPort] = useState('')
  const [initialLogLevel, setInitialLogLevel] = useState('')

  const handlePortChange = useCallback((value: string) => {
    if (!initialPort) setInitialPort(value)
    setPort(value)
  }, [initialPort])

  const handleLogLevelChange = useCallback((value: string) => {
    if (!initialLogLevel) setInitialLogLevel(value)
    setLogLevel(value)
  }, [initialLogLevel])

  const hasChanges = (port !== initialPort && initialPort !== '') ||
                     (logLevel !== initialLogLevel && initialLogLevel !== '')

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync({
        serverPort: parseInt(port),
        logLevel,
      })
      setInitialPort(port)
      setInitialLogLevel(logLevel)
      toast.success('Settings saved')
    } catch {
      toast.error('Failed to save settings')
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="System"
        description="Server configuration and authentication settings"
        breadcrumbs={[
          { label: 'Settings', href: '/settings/media' },
          { label: 'System' },
        ]}
        actions={
          <Button onClick={handleSave} disabled={updateMutation.isPending || !hasChanges}>
            <Save className="size-4 mr-2" />
            Save Changes
          </Button>
        }
      />

      <SystemNav />

      <div className="max-w-2xl">
        <ServerSection
          port={port}
          onPortChange={handlePortChange}
          logLevel={logLevel}
          onLogLevelChange={handleLogLevelChange}
        />
      </div>
    </div>
  )
}

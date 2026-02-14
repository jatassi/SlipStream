import { useEffect, useState } from 'react'

import { Bug, Loader2, TestTube } from 'lucide-react'
import { toast } from 'sonner'

import { downloadClientsApi } from '@/api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  useCreateDownloadClient,
  useDeveloperMode,
  useTestNewDownloadClient,
  useUpdateDownloadClient,
} from '@/hooks'
import type { CreateDownloadClientInput, DownloadClient, DownloadClientType } from '@/types'

type DownloadClientDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  client?: DownloadClient | null
}

type ClientTypeConfig = {
  label: string
  defaultPort: number
  supportsCategory: boolean
  usernameLabel: string
  passwordLabel: string
  usernameRequired: boolean
}

const clientTypeConfigs: Record<DownloadClientType, ClientTypeConfig> = {
  transmission: {
    label: 'Transmission',
    defaultPort: 9091,
    supportsCategory: false,
    usernameLabel: 'Username',
    passwordLabel: 'Password',
    usernameRequired: false,
  },
  qbittorrent: {
    label: 'qBittorrent',
    defaultPort: 8080,
    supportsCategory: true,
    usernameLabel: 'Username',
    passwordLabel: 'Password',
    usernameRequired: false,
  },
  sabnzbd: {
    label: 'SABnzbd',
    defaultPort: 8080,
    supportsCategory: true,
    usernameLabel: 'Username',
    passwordLabel: 'API Key',
    usernameRequired: false,
  },
  nzbget: {
    label: 'NZBGet',
    defaultPort: 6789,
    supportsCategory: true,
    usernameLabel: 'Username',
    passwordLabel: 'Password',
    usernameRequired: true,
  },
}

const defaultFormData: CreateDownloadClientInput = {
  name: '',
  type: 'transmission',
  host: 'localhost',
  port: 9091,
  username: '',
  password: '',
  useSsl: false,
  category: '',
  priority: 50,
  enabled: true,
}

export function DownloadClientDialog({ open, onOpenChange, client }: DownloadClientDialogProps) {
  const [formData, setFormData] = useState<CreateDownloadClientInput>(defaultFormData)
  const [isTesting, setIsTesting] = useState(false)
  const [isAddingDebugTorrent, setIsAddingDebugTorrent] = useState(false)

  const createMutation = useCreateDownloadClient()
  const updateMutation = useUpdateDownloadClient()
  const testNewMutation = useTestNewDownloadClient()
  const developerMode = useDeveloperMode()

  const isEditing = !!client

  useEffect(() => {
    if (open) {
      if (client) {
        setFormData({
          name: client.name,
          type: client.type,
          host: client.host,
          port: client.port,
          username: client.username || '',
          password: client.password || '',
          useSsl: client.useSsl,
          category: client.category || '',
          priority: client.priority,
          enabled: client.enabled,
        })
      } else {
        setFormData(defaultFormData)
      }
    }
  }, [open, client])

  const handleTypeChange = (type: DownloadClientType) => {
    const config = clientTypeConfigs[type]
    setFormData((prev) => ({
      ...prev,
      type,
      port: prev.port === clientTypeConfigs[prev.type].defaultPort ? config.defaultPort : prev.port,
    }))
  }

  const handleTest = async () => {
    setIsTesting(true)
    try {
      const result = await testNewMutation.mutateAsync(formData)
      if (result.success) {
        toast.success(result.message || 'Connection successful')
      } else {
        toast.error(result.message || 'Connection failed')
      }
    } catch {
      toast.error('Failed to test connection')
    } finally {
      setIsTesting(false)
    }
  }

  const handleDebugTorrent = async () => {
    if (!client) {
      return
    }

    setIsAddingDebugTorrent(true)
    try {
      const result = await downloadClientsApi.debugAddTorrent(client.id)
      if (result.success) {
        toast.success(result.message)
      } else {
        toast.error('Failed to add debug torrent')
      }
    } catch {
      toast.error('Failed to add debug torrent')
    } finally {
      setIsAddingDebugTorrent(false)
    }
  }

  const handleSubmit = async () => {
    if (!formData.name.trim()) {
      toast.error('Name is required')
      return
    }
    if (!formData.host.trim()) {
      toast.error('Host is required')
      return
    }

    try {
      if (isEditing && client) {
        await updateMutation.mutateAsync({
          id: client.id,
          data: formData,
        })
        toast.success('Client updated')
      } else {
        await createMutation.mutateAsync(formData)
        toast.success('Client added')
      }
      onOpenChange(false)
    } catch {
      toast.error(isEditing ? 'Failed to update client' : 'Failed to add client')
    }
  }

  const config = clientTypeConfigs[formData.type]
  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit Download Client' : 'Add Download Client'}</DialogTitle>
          <DialogDescription>
            Configure connection settings for your download client.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Client Type */}
          <div className="space-y-2">
            <Label htmlFor="type">Client Type</Label>
            <Select value={formData.type} onValueChange={(v) => handleTypeChange(v!)}>
              <SelectTrigger>
                <SelectValue>{clientTypeConfigs[formData.type].label}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="transmission">Transmission</SelectItem>
                <SelectItem value="qbittorrent">qBittorrent</SelectItem>
                <SelectItem value="sabnzbd">SABnzbd</SelectItem>
                <SelectItem value="nzbget">NZBGet</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Name */}
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              placeholder="My Download Client"
              value={formData.name}
              onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
            />
          </div>

          {/* Host & Port */}
          <div className="grid grid-cols-3 gap-4">
            <div className="col-span-2 space-y-2">
              <Label htmlFor="host">Host</Label>
              <Input
                id="host"
                placeholder="localhost"
                value={formData.host}
                onChange={(e) => setFormData((prev) => ({ ...prev, host: e.target.value }))}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="port">Port</Label>
              <Input
                id="port"
                type="number"
                value={formData.port}
                onChange={(e) =>
                  setFormData((prev) => ({ ...prev, port: Number.parseInt(e.target.value) || 0 }))
                }
              />
            </div>
          </div>

          {/* SSL Toggle */}
          <div className="flex items-center justify-between">
            <Label htmlFor="useSsl">Use SSL</Label>
            <Switch
              id="useSsl"
              checked={formData.useSsl}
              onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, useSsl: checked }))}
            />
          </div>

          {/* Username */}
          <div className="space-y-2">
            <Label htmlFor="username">
              {config.usernameLabel}
              {!config.usernameRequired && (
                <span className="text-muted-foreground ml-1 text-xs">(optional)</span>
              )}
            </Label>
            <Input
              id="username"
              value={formData.username}
              onChange={(e) => setFormData((prev) => ({ ...prev, username: e.target.value }))}
            />
          </div>

          {/* Password */}
          <div className="space-y-2">
            <Label htmlFor="password">
              {config.passwordLabel}
              <span className="text-muted-foreground ml-1 text-xs">(optional)</span>
            </Label>
            <Input
              id="password"
              type="password"
              value={formData.password}
              onChange={(e) => setFormData((prev) => ({ ...prev, password: e.target.value }))}
            />
          </div>

          {/* Category (only for clients that support it) */}
          {config.supportsCategory ? (
            <div className="space-y-2">
              <Label htmlFor="category">
                Category
                <span className="text-muted-foreground ml-1 text-xs">(optional)</span>
              </Label>
              <Input
                id="category"
                placeholder="slipstream"
                value={formData.category}
                onChange={(e) => setFormData((prev) => ({ ...prev, category: e.target.value }))}
              />
            </div>
          ) : null}

          {/* Priority */}
          <div className="space-y-2">
            <Label htmlFor="priority">Priority</Label>
            <Input
              id="priority"
              type="number"
              min={1}
              max={100}
              value={formData.priority}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  priority: Number.parseInt(e.target.value) || 50,
                }))
              }
            />
            <p className="text-muted-foreground text-xs">
              Lower values have higher priority (1-100)
            </p>
          </div>

          {/* Enabled Toggle */}
          <div className="flex items-center justify-between">
            <Label htmlFor="enabled">Enabled</Label>
            <Switch
              id="enabled"
              checked={formData.enabled}
              onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, enabled: checked }))}
            />
          </div>
        </div>

        <DialogFooter className="flex-col gap-2 sm:flex-row">
          <div className="flex gap-2">
            <Button variant="outline" onClick={handleTest} disabled={isTesting}>
              {isTesting ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <TestTube className="mr-2 size-4" />
              )}
              Test
            </Button>
            {developerMode && isEditing && formData.type === 'transmission' ? (
              <Button
                variant="outline"
                onClick={handleDebugTorrent}
                disabled={isAddingDebugTorrent}
                title="Add mock download for testing"
              >
                {isAddingDebugTorrent ? (
                  <Loader2 className="mr-2 size-4 animate-spin" />
                ) : (
                  <Bug className="mr-2 size-4" />
                )}
                Debug
              </Button>
            ) : null}
          </div>
          <div className="flex gap-2 sm:ml-auto">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button onClick={handleSubmit} disabled={isPending}>
              {isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
              {isEditing ? 'Save' : 'Add'}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

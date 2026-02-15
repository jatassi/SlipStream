import { useState } from 'react'

import { toast } from 'sonner'

import { downloadClientsApi } from '@/api'
import {
  useCreateDownloadClient,
  useDeveloperMode,
  useTestNewDownloadClient,
  useUpdateDownloadClient,
} from '@/hooks'
import type { CreateDownloadClientInput, DownloadClient, DownloadClientType } from '@/types'

type ClientTypeConfig = {
  label: string
  defaultPort: number
  supportsCategory: boolean
  usernameLabel: string
  passwordLabel: string
  usernameRequired: boolean
}

export const clientTypeConfigs: Record<DownloadClientType, ClientTypeConfig> = {
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

function createFormDataFromClient(client: DownloadClient): CreateDownloadClientInput {
  return {
    name: client.name,
    type: client.type,
    host: client.host,
    port: client.port,
    username: client.username ?? '',
    password: client.password ?? '',
    useSsl: client.useSsl,
    category: client.category ?? '',
    priority: client.priority,
    enabled: client.enabled,
  }
}

async function testClientConnection(
  formData: CreateDownloadClientInput,
  testNewMutation: ReturnType<typeof useTestNewDownloadClient>,
) {
  const result = await testNewMutation.mutateAsync(formData)
  if (result.success) {
    toast.success(result.message || 'Connection successful')
  } else {
    toast.error(result.message || 'Connection failed')
  }
}

async function addDebugTorrent(client: DownloadClient) {
  const result = await downloadClientsApi.debugAddTorrent(client.id)
  if (result.success) {
    toast.success(result.message)
  } else {
    toast.error('Failed to add debug torrent')
  }
}

type SubmitParams = {
  formData: CreateDownloadClientInput
  isEditing: boolean
  client: DownloadClient | null | undefined
  createMutation: ReturnType<typeof useCreateDownloadClient>
  updateMutation: ReturnType<typeof useUpdateDownloadClient>
}

async function submitClientForm(params: SubmitParams) {
  const { formData, isEditing, client, createMutation, updateMutation } = params

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
}

function useClientFormState(open: boolean, client: DownloadClient | null | undefined) {
  const [formData, setFormData] = useState<CreateDownloadClientInput>(defaultFormData)
  const [isTesting, setIsTesting] = useState(false)
  const [isAddingDebugTorrent, setIsAddingDebugTorrent] = useState(false)
  const [prevOpen, setPrevOpen] = useState(open)
  const [prevClient, setPrevClient] = useState(client)

  if (open !== prevOpen || client !== prevClient) {
    setPrevOpen(open); setPrevClient(client)
    if (open) { setFormData(client ? createFormDataFromClient(client) : defaultFormData) }
  }

  const handleTypeChange = (type: DownloadClientType) => {
    const config = clientTypeConfigs[type]
    setFormData((prev) => ({
      ...prev, type,
      port: prev.port === clientTypeConfigs[prev.type].defaultPort ? config.defaultPort : prev.port,
    }))
  }

  return { formData, setFormData, isTesting, setIsTesting, isAddingDebugTorrent, setIsAddingDebugTorrent, handleTypeChange }
}

export function useDownloadClientDialog(
  open: boolean,
  client: DownloadClient | null | undefined,
  onOpenChange: (open: boolean) => void,
) {
  const state = useClientFormState(open, client)
  const createMutation = useCreateDownloadClient()
  const updateMutation = useUpdateDownloadClient()
  const testNewMutation = useTestNewDownloadClient()
  const developerMode = useDeveloperMode()
  const isEditing = !!client

  const handleTest = async () => {
    state.setIsTesting(true)
    try { await testClientConnection(state.formData, testNewMutation) }
    catch { toast.error('Failed to test connection') }
    finally { state.setIsTesting(false) }
  }

  const handleDebugTorrent = async () => {
    if (!client) { return }
    state.setIsAddingDebugTorrent(true)
    try { await addDebugTorrent(client) }
    catch { toast.error('Failed to add debug torrent') }
    finally { state.setIsAddingDebugTorrent(false) }
  }

  const handleSubmit = async () => {
    if (!state.formData.name.trim()) { toast.error('Name is required'); return }
    if (!state.formData.host.trim()) { toast.error('Host is required'); return }
    try {
      await submitClientForm({ formData: state.formData, isEditing, client, createMutation, updateMutation })
      onOpenChange(false)
    } catch { toast.error(isEditing ? 'Failed to update client' : 'Failed to add client') }
  }

  return {
    ...state, developerMode, isEditing,
    isPending: createMutation.isPending || updateMutation.isPending,
    config: clientTypeConfigs[state.formData.type],
    handleTest, handleDebugTorrent, handleSubmit,
  }
}

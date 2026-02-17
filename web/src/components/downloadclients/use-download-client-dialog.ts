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
  defaultUrlBase: string
  defaultSsl: boolean
  supportsCategory: boolean
  supportsUrlBase: boolean
  supportsApiKey: boolean
  supportsUsername: boolean
  supportsPassword: boolean
  usernameLabel: string
  passwordLabel: string
  apiKeyLabel: string
  passwordRequired: boolean
}

const configDefaults: ClientTypeConfig = {
  label: '', defaultPort: 8080, defaultUrlBase: '/', defaultSsl: false,
  supportsCategory: false, supportsUrlBase: true, supportsApiKey: false,
  supportsUsername: true, supportsPassword: true,
  usernameLabel: 'Username', passwordLabel: 'Password', apiKeyLabel: '', passwordRequired: false,
}

function cfg(overrides: Partial<ClientTypeConfig> & Pick<ClientTypeConfig, 'label' | 'defaultPort'>): ClientTypeConfig {
  return { ...configDefaults, ...overrides }
}

export const clientTypeConfigs: Record<DownloadClientType, ClientTypeConfig> = {
  transmission: cfg({ label: 'Transmission', defaultPort: 9091, defaultUrlBase: '/transmission/' }),
  qbittorrent: cfg({ label: 'qBittorrent', defaultPort: 8080, supportsCategory: true, supportsApiKey: true, apiKeyLabel: 'API Key' }),
  deluge: cfg({ label: 'Deluge', defaultPort: 8112, supportsCategory: true, supportsUsername: false, passwordRequired: true }),
  rtorrent: cfg({ label: 'rTorrent', defaultPort: 8080, defaultUrlBase: '/RPC2', supportsCategory: true }),
  vuze: cfg({ label: 'Vuze', defaultPort: 9091, defaultUrlBase: '/transmission/' }),
  flood: cfg({ label: 'Flood', defaultPort: 3000, supportsCategory: true, passwordRequired: true }),
  aria2: cfg({ label: 'Aria2', defaultPort: 6800, defaultUrlBase: '/jsonrpc', supportsApiKey: true, supportsUsername: false, supportsPassword: false, usernameLabel: '', passwordLabel: '', apiKeyLabel: 'Secret Token' }),
  utorrent: cfg({ label: 'uTorrent', defaultPort: 8080, defaultUrlBase: '/gui/', supportsCategory: true }),
  hadouken: cfg({ label: 'Hadouken', defaultPort: 7070, supportsCategory: true }),
  downloadstation: cfg({ label: 'Download Station', defaultPort: 5000, supportsUrlBase: false, passwordRequired: true }),
  freeboxdownload: cfg({ label: 'Freebox Download', defaultPort: 443, defaultUrlBase: '/api/v1/', defaultSsl: true, supportsApiKey: true, supportsUsername: false, supportsPassword: false, usernameLabel: '', passwordLabel: '', apiKeyLabel: 'App Token' }),
  rqbit: cfg({ label: 'rqbit', defaultPort: 3030, supportsUsername: false, supportsPassword: false, usernameLabel: '', passwordLabel: '' }),
  tribler: cfg({ label: 'Tribler', defaultPort: 20_100, supportsApiKey: true, supportsUsername: false, supportsPassword: false, usernameLabel: '', passwordLabel: '', apiKeyLabel: 'API Key' }),
}

const defaultFormData: CreateDownloadClientInput = {
  name: '',
  type: 'transmission',
  host: 'localhost',
  port: 9091,
  username: '',
  password: '',
  useSsl: false,
  apiKey: '',
  category: '',
  urlBase: '/transmission/',
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
    apiKey: client.apiKey ?? '',
    category: client.category ?? '',
    urlBase: client.urlBase ?? '',
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
    const prevConfig = clientTypeConfigs[formData.type]
    setFormData((prev) => ({
      ...prev, type,
      port: prev.port === prevConfig.defaultPort ? config.defaultPort : prev.port,
      urlBase: prev.urlBase === prevConfig.defaultUrlBase ? config.defaultUrlBase : prev.urlBase,
      useSsl: prev.useSsl === prevConfig.defaultSsl ? config.defaultSsl : prev.useSsl,
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

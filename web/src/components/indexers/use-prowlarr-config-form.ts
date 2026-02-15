import { useState } from 'react'

import { toast } from 'sonner'

import {
  useProwlarrConfig,
  useProwlarrStatus,
  useRefreshProwlarr,
  useTestProwlarrConnection,
  useUpdateProwlarrConfig,
} from '@/hooks'
import type { ProwlarrConfigInput, ProwlarrTestInput } from '@/types'
import { DEFAULT_MOVIE_CATEGORIES, DEFAULT_TV_CATEGORIES } from '@/types'

function parseUrl(url: string): { hostname: string; port: string; useSsl: boolean } {
  if (!url) {
    return { hostname: '', port: '9696', useSsl: false }
  }
  try {
    const parsed = new URL(url)
    const useSsl = parsed.protocol === 'https:'
    const defaultPort = useSsl ? '443' : '9696'
    return {
      hostname: parsed.hostname,
      port: parsed.port || defaultPort,
      useSsl,
    }
  } catch {
    return { hostname: url, port: '9696', useSsl: false }
  }
}

function buildUrl(hostname: string, port: string, useSsl: boolean): string {
  if (!hostname) {
    return ''
  }
  const protocol = useSsl ? 'https' : 'http'
  const defaultPort = useSsl ? '443' : '80'
  const portSuffix = port && port !== defaultPort ? `:${port}` : ''
  return `${protocol}://${hostname}${portSuffix}`
}

type TestParams = {
  hostname: string
  apiKey: string
  port: string
  useSsl: boolean
  timeout: number
  skipSslVerify: boolean
  testMutation: ReturnType<typeof useTestProwlarrConnection>
}

type SaveParams = {
  hostname: string
  apiKey: string
  port: string
  useSsl: boolean
  timeout: number
  skipSslVerify: boolean
  movieCategories: number[]
  tvCategories: number[]
  updateMutation: ReturnType<typeof useUpdateProwlarrConfig>
}

async function testProwlarrConnection(params: TestParams) {
  const { hostname, apiKey, port, useSsl, timeout, skipSslVerify, testMutation } = params
  const url = buildUrl(hostname, port, useSsl)
  const testInput: ProwlarrTestInput = {
    url,
    apiKey,
    timeout,
    skipSslVerify,
  }

  const result = await testMutation.mutateAsync(testInput)
  if (result.success) {
    toast.success('Connection successful')
  } else {
    toast.error(result.message ?? 'Connection failed')
  }
}

async function saveProwlarrConfig(params: SaveParams) {
  const {
    hostname,
    apiKey,
    port,
    useSsl,
    timeout,
    skipSslVerify,
    movieCategories,
    tvCategories,
    updateMutation,
  } = params
  const url = buildUrl(hostname, port, useSsl)
  const configInput: ProwlarrConfigInput = {
    enabled: true,
    url,
    apiKey,
    timeout,
    skipSslVerify,
    movieCategories,
    tvCategories,
  }

  await updateMutation.mutateAsync(configInput)
  toast.success('Configuration saved')
}

function useProwlarrFormState() {
  const { data: config, isLoading: configLoading } = useProwlarrConfig()
  const { data: status } = useProwlarrStatus()
  const [hostname, setHostname] = useState('')
  const [port, setPort] = useState('9696')
  const [useSsl, setUseSsl] = useState(false)
  const [apiKey, setApiKey] = useState('')
  const [timeout, setTimeout] = useState(30)
  const [skipSslVerify, setSkipSslVerify] = useState(false)
  const [movieCategories, setMovieCategories] = useState<number[]>(DEFAULT_MOVIE_CATEGORIES)
  const [tvCategories, setTvCategories] = useState<number[]>(DEFAULT_TV_CATEGORIES)
  const [showApiKey, setShowApiKey] = useState(false)
  const [isDirty, setIsDirty] = useState(false)
  const [isExpanded, setIsExpanded] = useState(true)
  const [prevConfig, setPrevConfig] = useState(config)

  if (config !== prevConfig) {
    setPrevConfig(config)
    if (config) {
      const parsed = parseUrl(config.url || '')
      setHostname(parsed.hostname)
      setPort(parsed.port)
      setUseSsl(parsed.useSsl)
      setApiKey(config.apiKey || '')
      setTimeout(config.timeout || 30)
      setSkipSslVerify(config.skipSslVerify || false)
      setMovieCategories(config.movieCategories.length > 0 ? config.movieCategories : DEFAULT_MOVIE_CATEGORIES)
      setTvCategories(config.tvCategories.length > 0 ? config.tvCategories : DEFAULT_TV_CATEGORIES)
      setIsDirty(false)
    }
  }

  const handleFieldChange = () => { setIsDirty(true) }

  const toggleCategory = (category: number, type: 'movie' | 'tv') => {
    handleFieldChange()
    const setter = type === 'movie' ? setMovieCategories : setTvCategories
    setter((prev) => prev.includes(category) ? prev.filter((c) => c !== category) : [...prev, category])
  }

  return {
    configLoading, status, hostname, setHostname, port, setPort, useSsl, setUseSsl,
    apiKey, setApiKey, timeout, setTimeout, skipSslVerify, setSkipSslVerify,
    movieCategories, tvCategories, showApiKey, setShowApiKey,
    isDirty, setIsDirty, isExpanded, setIsExpanded, handleFieldChange, toggleCategory,
  }
}

export function useProwlarrConfigForm() {
  const state = useProwlarrFormState()
  const updateMutation = useUpdateProwlarrConfig()
  const testMutation = useTestProwlarrConnection()
  const refreshMutation = useRefreshProwlarr()

  const handleTest = async () => {
    if (!state.hostname || !state.apiKey) { toast.error('Hostname and API Key are required'); return }
    try {
      await testProwlarrConnection({ hostname: state.hostname, apiKey: state.apiKey, port: state.port, useSsl: state.useSsl, timeout: state.timeout, skipSslVerify: state.skipSslVerify, testMutation })
    } catch { toast.error('Failed to test connection') }
  }

  const handleSave = async () => {
    if (!state.hostname || !state.apiKey) { toast.error('Hostname and API Key are required'); return }
    try {
      await saveProwlarrConfig({ hostname: state.hostname, apiKey: state.apiKey, port: state.port, useSsl: state.useSsl, timeout: state.timeout, skipSslVerify: state.skipSslVerify, movieCategories: state.movieCategories, tvCategories: state.tvCategories, updateMutation })
      state.setIsDirty(false)
    } catch { toast.error('Failed to save configuration') }
  }

  const handleRefresh = async () => {
    try { await refreshMutation.mutateAsync(); toast.success('Prowlarr data refreshed') }
    catch { toast.error('Failed to refresh Prowlarr data') }
  }

  return {
    ...state, testMutation, updateMutation, refreshMutation,
    handleTest, handleSave, handleRefresh,
  }
}

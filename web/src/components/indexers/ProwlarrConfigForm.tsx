import { useState } from 'react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from '@/components/ui/collapsible'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Loader2, TestTube, Save, RefreshCw, CheckCircle2, XCircle, Eye, EyeOff, ChevronDown } from 'lucide-react'
import {
  useProwlarrConfig,
  useUpdateProwlarrConfig,
  useTestProwlarrConnection,
  useProwlarrStatus,
  useRefreshProwlarr,
} from '@/hooks'
import { toast } from 'sonner'
import {
  DEFAULT_MOVIE_CATEGORIES,
  DEFAULT_TV_CATEGORIES,
  getCategoryName,
} from '@/types'
import type { ProwlarrConfigInput, ProwlarrTestInput } from '@/types'

// Parse URL into hostname and port
function parseUrl(url: string): { hostname: string; port: string; useSsl: boolean } {
  if (!url) return { hostname: '', port: '9696', useSsl: false }
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

// Build URL from hostname and port
function buildUrl(hostname: string, port: string, useSsl: boolean): string {
  if (!hostname) return ''
  const protocol = useSsl ? 'https' : 'http'
  const defaultPort = useSsl ? '443' : '80'
  const portSuffix = port && port !== defaultPort ? `:${port}` : ''
  return `${protocol}://${hostname}${portSuffix}`
}

export function ProwlarrConfigForm() {
  const { data: config, isLoading: configLoading } = useProwlarrConfig()
  const { data: status } = useProwlarrStatus()
  const updateMutation = useUpdateProwlarrConfig()
  const testMutation = useTestProwlarrConnection()
  const refreshMutation = useRefreshProwlarr()

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
  const [isExpanded, setIsExpanded] = useState(false)
  const [prevConfig, setPrevConfig] = useState(config)

  // Reset form when config changes (React-recommended pattern)
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
      setMovieCategories(config.movieCategories?.length ? config.movieCategories : DEFAULT_MOVIE_CATEGORIES)
      setTvCategories(config.tvCategories?.length ? config.tvCategories : DEFAULT_TV_CATEGORIES)
      setIsDirty(false)
    }
  }

  const handleFieldChange = () => {
    setIsDirty(true)
  }

  const handleTest = async () => {
    if (!hostname || !apiKey) {
      toast.error('Hostname and API Key are required')
      return
    }

    const url = buildUrl(hostname, port, useSsl)
    const testInput: ProwlarrTestInput = {
      url,
      apiKey,
      timeout,
      skipSslVerify,
    }

    try {
      const result = await testMutation.mutateAsync(testInput)
      if (result.success) {
        toast.success('Connection successful')
      } else {
        toast.error(result.message || 'Connection failed')
      }
    } catch {
      toast.error('Failed to test connection')
    }
  }

  const handleSave = async () => {
    if (!hostname || !apiKey) {
      toast.error('Hostname and API Key are required')
      return
    }

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

    try {
      await updateMutation.mutateAsync(configInput)
      toast.success('Configuration saved')
      setIsDirty(false)
    } catch {
      toast.error('Failed to save configuration')
    }
  }

  const handleRefresh = async () => {
    try {
      await refreshMutation.mutateAsync()
      toast.success('Prowlarr data refreshed')
    } catch {
      toast.error('Failed to refresh Prowlarr data')
    }
  }

  const toggleCategory = (category: number, type: 'movie' | 'tv') => {
    handleFieldChange()
    if (type === 'movie') {
      setMovieCategories((prev) =>
        prev.includes(category) ? prev.filter((c) => c !== category) : [...prev, category]
      )
    } else {
      setTvCategories((prev) =>
        prev.includes(category) ? prev.filter((c) => c !== category) : [...prev, category]
      )
    }
  }

  if (configLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <Collapsible open={isExpanded} onOpenChange={setIsExpanded}>
        <CardHeader>
          <CollapsibleTrigger className="flex items-center justify-between w-full text-left">
            <div>
              <CardTitle className="text-base">Prowlarr Configuration</CardTitle>
              <CardDescription>
                Connect to your Prowlarr instance for centralized indexer management
              </CardDescription>
            </div>
            <div className="flex items-center gap-3">
              {status && (
                <Badge variant={status.connected ? 'default' : 'destructive'} className="gap-1">
                  {status.connected ? (
                    <>
                      <CheckCircle2 className="size-3" />
                      Connected {status.version && `(v${status.version})`}
                    </>
                  ) : (
                    <>
                      <XCircle className="size-3" />
                      Disconnected
                    </>
                  )}
                </Badge>
              )}
              <ChevronDown className={`size-5 text-muted-foreground transition-transform ${isExpanded ? 'rotate-180' : ''}`} />
            </div>
          </CollapsibleTrigger>
        </CardHeader>
        <CollapsibleContent>
          <CardContent className="space-y-6 pt-0">
            <div className="grid gap-4">
          <div className="grid gap-2">
            <Label>Host</Label>
            <div className="flex gap-0">
              <Select value={useSsl ? 'https' : 'http'} onValueChange={(v) => { setUseSsl(v === 'https'); handleFieldChange() }}>
                <SelectTrigger className="w-[100px] rounded-r-none border-r-0">
                  {useSsl ? 'https://' : 'http://'}
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="http">http://</SelectItem>
                  <SelectItem value="https">https://</SelectItem>
                </SelectContent>
              </Select>
              <Input
                id="prowlarr-hostname"
                placeholder="localhost"
                className="rounded-l-none rounded-r-none flex-1"
                value={hostname}
                onChange={(e) => {
                  setHostname(e.target.value)
                  handleFieldChange()
                }}
              />
              <div className="flex items-center border border-l-0 rounded-r-md px-2 bg-muted text-muted-foreground text-sm">:</div>
              <Input
                id="prowlarr-port"
                type="number"
                className="w-20 rounded-l-none"
                placeholder="9696"
                value={port}
                onChange={(e) => {
                  setPort(e.target.value)
                  handleFieldChange()
                }}
              />
            </div>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="prowlarr-apikey">API Key</Label>
            <div className="relative">
              <Input
                id="prowlarr-apikey"
                type={showApiKey ? 'text' : 'password'}
                placeholder="Enter your Prowlarr API key"
                value={apiKey}
                onChange={(e) => {
                  setApiKey(e.target.value)
                  handleFieldChange()
                }}
                className="pr-10"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                onClick={() => setShowApiKey(!showApiKey)}
              >
                {showApiKey ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              Found in Prowlarr under Settings → General → Security
            </p>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="prowlarr-timeout">Timeout (seconds)</Label>
            <Input
              id="prowlarr-timeout"
              type="number"
              min={5}
              max={300}
              value={timeout}
              onChange={(e) => {
                setTimeout(parseInt(e.target.value) || 30)
                handleFieldChange()
              }}
            />
          </div>

          {useSsl && (
            <div className="flex items-center gap-2">
              <Checkbox
                id="prowlarr-skip-ssl"
                checked={skipSslVerify}
                onCheckedChange={(checked) => {
                  setSkipSslVerify(checked === true)
                  handleFieldChange()
                }}
              />
              <Label htmlFor="prowlarr-skip-ssl" className="cursor-pointer">
                Skip SSL certificate verification
              </Label>
            </div>
          )}
        </div>

        <div className="space-y-4">
          <div>
            <Label className="text-sm font-medium">Movie Categories</Label>
            <p className="text-xs text-muted-foreground mb-2">
              Newznab category IDs to search for movies
            </p>
            <div className="flex flex-wrap gap-2">
              {DEFAULT_MOVIE_CATEGORIES.map((cat) => (
                <Badge
                  key={cat}
                  variant={movieCategories.includes(cat) ? 'default' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => toggleCategory(cat, 'movie')}
                >
                  {getCategoryName(cat)}
                </Badge>
              ))}
            </div>
          </div>

          <div>
            <Label className="text-sm font-medium">TV Categories</Label>
            <p className="text-xs text-muted-foreground mb-2">
              Newznab category IDs to search for TV shows
            </p>
            <div className="flex flex-wrap gap-2">
              {DEFAULT_TV_CATEGORIES.map((cat) => (
                <Badge
                  key={cat}
                  variant={tvCategories.includes(cat) ? 'default' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => toggleCategory(cat, 'tv')}
                >
                  {getCategoryName(cat)}
                </Badge>
              ))}
            </div>
          </div>
        </div>
          </CardContent>
        </CollapsibleContent>
      </Collapsible>
      <CardContent className="pt-0">
        <div className="flex items-center gap-2 pt-4 border-t">
          <Button
            onClick={handleTest}
            variant="outline"
            disabled={testMutation.isPending || !hostname || !apiKey}
          >
            {testMutation.isPending ? (
              <Loader2 className="size-4 mr-2 animate-spin" />
            ) : (
              <TestTube className="size-4 mr-2" />
            )}
            Test
          </Button>
          <Button
            onClick={handleRefresh}
            variant="outline"
            disabled={refreshMutation.isPending || !status?.connected}
          >
            {refreshMutation.isPending ? (
              <Loader2 className="size-4 mr-2 animate-spin" />
            ) : (
              <RefreshCw className="size-4 mr-2" />
            )}
            Refresh
          </Button>
          <Button
            onClick={handleSave}
            disabled={updateMutation.isPending || !isDirty || !hostname || !apiKey}
          >
            {updateMutation.isPending ? (
              <Loader2 className="size-4 mr-2 animate-spin" />
            ) : (
              <Save className="size-4 mr-2" />
            )}
            Save
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

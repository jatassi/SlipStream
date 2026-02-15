import {
  CheckCircle2,
  ChevronDown,
  Eye,
  EyeOff,
  Loader2,
  RefreshCw,
  Save,
  TestTube,
  XCircle,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { LoadingButton } from '@/components/ui/loading-button'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { DEFAULT_MOVIE_CATEGORIES, DEFAULT_TV_CATEGORIES, getCategoryName } from '@/types'

import { useProwlarrConfigForm } from './use-prowlarr-config-form'

export function ProwlarrConfigForm() {
  const hook = useProwlarrConfigForm()

  if (hook.configLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Loader2 className="text-muted-foreground size-6 animate-spin" />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <Collapsible open={hook.isExpanded} onOpenChange={hook.setIsExpanded}>
        <ConfigHeader status={hook.status} isExpanded={hook.isExpanded} />
        <CollapsibleContent>
          <CardContent className="space-y-6 pt-0">
            <ConnectionSettings hook={hook} />
            <CategorySettings hook={hook} />
          </CardContent>
        </CollapsibleContent>
      </Collapsible>
      <ActionButtons hook={hook} />
    </Card>
  )
}

function ConfigHeader({
  status,
  isExpanded,
}: {
  status: { connected: boolean; version?: string } | undefined
  isExpanded: boolean
}) {
  return (
    <CardHeader>
      <CollapsibleTrigger className="flex w-full items-center justify-between text-left">
        <div>
          <CardTitle className="text-base">Prowlarr Configuration</CardTitle>
          <CardDescription>
            Connect to your Prowlarr instance for centralized indexer management
          </CardDescription>
        </div>
        <div className="flex items-center gap-3">
          {status ? (
            <Badge variant={status.connected ? 'default' : 'destructive'} className="gap-1">
              {status.connected ? (
                <>
                  <CheckCircle2 className="size-3" />
                  Connected {status.version ? `(v${status.version})` : null}
                </>
              ) : (
                <>
                  <XCircle className="size-3" />
                  Disconnected
                </>
              )}
            </Badge>
          ) : null}
          <ChevronDown
            className={`text-muted-foreground size-5 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
          />
        </div>
      </CollapsibleTrigger>
    </CardHeader>
  )
}

type HookValues = ReturnType<typeof useProwlarrConfigForm>

function ConnectionSettings({ hook }: { hook: HookValues }) {
  return (
    <div className="grid gap-4">
      <HostInput hook={hook} />
      <ApiKeyInput hook={hook} />
      <TimeoutInput hook={hook} />
      {hook.useSsl ? <SslVerifyCheckbox hook={hook} /> : null}
    </div>
  )
}

function HostInput({ hook }: { hook: HookValues }) {
  return (
    <div className="grid gap-2">
      <Label>Host</Label>
      <div className="flex gap-0">
        <Select
          value={hook.useSsl ? 'https' : 'http'}
          onValueChange={(v) => {
            hook.setUseSsl(v === 'https')
            hook.handleFieldChange()
          }}
        >
          <SelectTrigger className="w-[100px] rounded-r-none border-r-0">
            {hook.useSsl ? 'https://' : 'http://'}
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="http">http://</SelectItem>
            <SelectItem value="https">https://</SelectItem>
          </SelectContent>
        </Select>
        <Input
          id="prowlarr-hostname"
          placeholder="localhost"
          className="flex-1 rounded-l-none rounded-r-none"
          value={hook.hostname}
          onChange={(e) => {
            hook.setHostname(e.target.value)
            hook.handleFieldChange()
          }}
        />
        <div className="bg-muted text-muted-foreground flex items-center rounded-r-md border border-l-0 px-2 text-sm">
          :
        </div>
        <Input
          id="prowlarr-port"
          type="number"
          className="w-20 rounded-l-none"
          placeholder="9696"
          value={hook.port}
          onChange={(e) => {
            hook.setPort(e.target.value)
            hook.handleFieldChange()
          }}
        />
      </div>
    </div>
  )
}

function ApiKeyInput({ hook }: { hook: HookValues }) {
  return (
    <div className="grid gap-2">
      <Label htmlFor="prowlarr-apikey">API Key</Label>
      <div className="relative">
        <Input
          id="prowlarr-apikey"
          type={hook.showApiKey ? 'text' : 'password'}
          placeholder="Enter your Prowlarr API key"
          value={hook.apiKey}
          onChange={(e) => {
            hook.setApiKey(e.target.value)
            hook.handleFieldChange()
          }}
          className="pr-10"
        />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="absolute top-0 right-0 h-full px-3 hover:bg-transparent"
          onClick={() => hook.setShowApiKey(!hook.showApiKey)}
        >
          {hook.showApiKey ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
        </Button>
      </div>
      <p className="text-muted-foreground text-xs">
        Found in Prowlarr under Settings → General → Security
      </p>
    </div>
  )
}

function TimeoutInput({ hook }: { hook: HookValues }) {
  return (
    <div className="grid gap-2">
      <Label htmlFor="prowlarr-timeout">Timeout (seconds)</Label>
      <Input
        id="prowlarr-timeout"
        type="number"
        min={5}
        max={300}
        value={hook.timeout}
        onChange={(e) => {
          hook.setTimeout(Number.parseInt(e.target.value) || 30)
          hook.handleFieldChange()
        }}
      />
    </div>
  )
}

function SslVerifyCheckbox({ hook }: { hook: HookValues }) {
  return (
    <div className="flex items-center gap-2">
      <Checkbox
        id="prowlarr-skip-ssl"
        checked={hook.skipSslVerify}
        onCheckedChange={(checked) => {
          hook.setSkipSslVerify(checked)
          hook.handleFieldChange()
        }}
      />
      <Label htmlFor="prowlarr-skip-ssl" className="cursor-pointer">
        Skip SSL certificate verification
      </Label>
    </div>
  )
}

function CategorySettings({ hook }: { hook: HookValues }) {
  return (
    <div className="space-y-4">
      <div>
        <Label className="text-sm font-medium">Movie Categories</Label>
        <p className="text-muted-foreground mb-2 text-xs">
          Newznab category IDs to search for movies
        </p>
        <div className="flex flex-wrap gap-2">
          {DEFAULT_MOVIE_CATEGORIES.map((cat) => (
            <Badge
              key={cat}
              variant={hook.movieCategories.includes(cat) ? 'default' : 'outline'}
              className="cursor-pointer"
              onClick={() => hook.toggleCategory(cat, 'movie')}
            >
              {getCategoryName(cat)}
            </Badge>
          ))}
        </div>
      </div>

      <div>
        <Label className="text-sm font-medium">TV Categories</Label>
        <p className="text-muted-foreground mb-2 text-xs">
          Newznab category IDs to search for TV shows
        </p>
        <div className="flex flex-wrap gap-2">
          {DEFAULT_TV_CATEGORIES.map((cat) => (
            <Badge
              key={cat}
              variant={hook.tvCategories.includes(cat) ? 'default' : 'outline'}
              className="cursor-pointer"
              onClick={() => hook.toggleCategory(cat, 'tv')}
            >
              {getCategoryName(cat)}
            </Badge>
          ))}
        </div>
      </div>
    </div>
  )
}

function ActionButtons({ hook }: { hook: HookValues }) {
  const testDisabled = hook.testMutation.isPending || !hook.hostname || !hook.apiKey
  const refreshDisabled = hook.refreshMutation.isPending || !hook.status?.connected
  const saveDisabled = hook.updateMutation.isPending || !hook.isDirty || !hook.hostname || !hook.apiKey
  return (
    <CardContent className="pt-0">
      <div className="flex items-center gap-2 border-t pt-4">
        <LoadingButton loading={hook.testMutation.isPending} icon={TestTube} variant="outline" onClick={hook.handleTest} disabled={testDisabled}>
          Test
        </LoadingButton>
        <LoadingButton loading={hook.refreshMutation.isPending} icon={RefreshCw} variant="outline" onClick={hook.handleRefresh} disabled={refreshDisabled}>
          Refresh
        </LoadingButton>
        <LoadingButton loading={hook.updateMutation.isPending} icon={Save} onClick={hook.handleSave} disabled={saveDisabled}>
          Save
        </LoadingButton>
      </div>
    </CardContent>
  )
}

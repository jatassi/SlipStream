import type { UseQueryResult } from '@tanstack/react-query'
import { CheckCircle2, FolderOpen, Loader2 } from 'lucide-react'

import { FolderBrowser } from '@/components/forms/folder-browser'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import type { DetectDBResponse, SourceType } from '@/types/arr-import'

import { type ConnectionMethod, useConnectStep } from './use-connect-step'

type ConnectStepProps = {
  sourceType: SourceType
  onSourceTypeChange: (type: SourceType) => void
  onConnected: () => void
}

export function ConnectStep({ sourceType, onSourceTypeChange, onConnected }: ConnectStepProps) {
  const {
    connectionMethod,
    setConnectionMethod,
    dbPath,
    setDbPath,
    url,
    setUrl,
    apiKey,
    setApiKey,
    browserOpen,
    setBrowserOpen,
    connectMutation,
    handleConnect,
    isValid,
    detectQuery,
  } = useConnectStep(sourceType, onConnected)

  return (
    <div className="space-y-6">
      <SourceTypeSelector sourceType={sourceType} onChange={onSourceTypeChange} />
      <ConnectionMethodSelector method={connectionMethod} onChange={setConnectionMethod} />
      <ConnectionFields
        connectionMethod={connectionMethod}
        dbPath={dbPath}
        onDbPathChange={setDbPath}
        onBrowseClick={() => setBrowserOpen(true)}
        url={url}
        onUrlChange={setUrl}
        apiKey={apiKey}
        onApiKeyChange={setApiKey}
        detectQuery={detectQuery}
      />
      <ConnectionError error={connectMutation.error} />
      <div className="flex justify-end">
        <Button onClick={handleConnect} disabled={!isValid || connectMutation.isPending}>
          {connectMutation.isPending ? 'Connecting...' : 'Connect'}
        </Button>
      </div>
      <FolderBrowser
        open={browserOpen}
        onOpenChange={setBrowserOpen}
        initialPath={dbPath}
        onSelect={setDbPath}
      />
    </div>
  )
}

function SourceTypeSelector({
  sourceType,
  onChange,
}: {
  sourceType: SourceType
  onChange: (type: SourceType) => void
}) {
  return (
    <div className="space-y-3">
      <Label>Source Application</Label>
      <RadioGroup value={sourceType} onValueChange={(v) => onChange(v as SourceType)}>
        <div className="flex items-center gap-2">
          <RadioGroupItem value="radarr" />
          <Label>Radarr</Label>
        </div>
        <div className="flex items-center gap-2">
          <RadioGroupItem value="sonarr" />
          <Label>Sonarr</Label>
        </div>
      </RadioGroup>
    </div>
  )
}

function ConnectionMethodSelector({
  method,
  onChange,
}: {
  method: ConnectionMethod
  onChange: (method: ConnectionMethod) => void
}) {
  return (
    <div className="space-y-3">
      <Label>Connection Method</Label>
      <RadioGroup value={method} onValueChange={(v) => onChange(v as ConnectionMethod)}>
        <div className="flex items-center gap-2">
          <RadioGroupItem value="sqlite" />
          <Label>SQLite Database</Label>
        </div>
        <div className="flex items-center gap-2">
          <RadioGroupItem value="api" />
          <Label>API Connection</Label>
        </div>
      </RadioGroup>
    </div>
  )
}

function ConnectionFields({
  connectionMethod,
  dbPath,
  onDbPathChange,
  onBrowseClick,
  url,
  onUrlChange,
  apiKey,
  onApiKeyChange,
  detectQuery,
}: {
  connectionMethod: ConnectionMethod
  dbPath: string
  onDbPathChange: (path: string) => void
  onBrowseClick: () => void
  url: string
  onUrlChange: (url: string) => void
  apiKey: string
  onApiKeyChange: (key: string) => void
  detectQuery: UseQueryResult<DetectDBResponse>
}) {
  if (connectionMethod === 'sqlite') {
    return (
      <SqliteFields
        dbPath={dbPath}
        onDbPathChange={onDbPathChange}
        onBrowseClick={onBrowseClick}
        detectQuery={detectQuery}
      />
    )
  }

  return <ApiFields url={url} onUrlChange={onUrlChange} apiKey={apiKey} onApiKeyChange={onApiKeyChange} />
}

function ConnectionError({ error }: { error: Error | null }) {
  if (!error) {
    return null
  }

  return (
    <div className="text-destructive rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm">
      {error.message}
    </div>
  )
}

function SqliteFields({
  dbPath,
  onDbPathChange,
  onBrowseClick,
  detectQuery,
}: {
  dbPath: string
  onDbPathChange: (path: string) => void
  onBrowseClick: () => void
  detectQuery: UseQueryResult<DetectDBResponse>
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor="db-path">Database Path</Label>
      <div className="flex gap-2">
        <Input
          id="db-path"
          value={dbPath}
          onChange={(e) => onDbPathChange(e.target.value)}
          placeholder="/path/to/radarr.db"
        />
        <Button
          variant="outline"
          size="icon"
          onClick={onBrowseClick}
          type="button"
          data-icon="inline-end"
        >
          <FolderOpen />
        </Button>
      </div>
      <DetectStatus detectQuery={detectQuery} />
    </div>
  )
}

function DetectStatus({ detectQuery }: { detectQuery: UseQueryResult<DetectDBResponse> }) {
  if (detectQuery.isLoading) {
    return (
      <p className="text-muted-foreground flex items-center gap-1.5 text-sm">
        <Loader2 className="size-3.5 animate-spin" />
        Detecting database location...
      </p>
    )
  }

  if (detectQuery.data?.found) {
    return (
      <p className="flex items-center gap-1.5 text-sm text-green-500">
        <CheckCircle2 className="size-3.5" />
        Database found at default location
      </p>
    )
  }

  if (detectQuery.isSuccess) {
    return (
      <p className="text-muted-foreground text-sm">
        Database not found at default locations. Enter the path manually or browse to select it.
      </p>
    )
  }

  return null
}

function ApiFields({
  url,
  onUrlChange,
  apiKey,
  onApiKeyChange,
}: {
  url: string
  onUrlChange: (url: string) => void
  apiKey: string
  onApiKeyChange: (key: string) => void
}) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="url">URL</Label>
        <Input
          id="url"
          value={url}
          onChange={(e) => onUrlChange(e.target.value)}
          placeholder="http://localhost:7878"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="api-key">API Key</Label>
        <Input
          id="api-key"
          type="password"
          value={apiKey}
          onChange={(e) => onApiKeyChange(e.target.value)}
          placeholder="API Key"
        />
      </div>
    </>
  )
}

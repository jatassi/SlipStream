import { AlertTriangle, ExternalLink, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { useIndexerMode, useSetIndexerMode } from '@/hooks'
import type { IndexerMode } from '@/types'

type IndexerModeToggleProps = {
  onModeChange?: (mode: IndexerMode) => void
}

export function IndexerModeToggle({ onModeChange }: IndexerModeToggleProps) {
  const { data: modeInfo, isLoading } = useIndexerMode()
  const setModeMutation = useSetIndexerMode()

  const handleModeChange = async (mode: IndexerMode) => {
    try {
      await setModeMutation.mutateAsync({ mode })
      toast.success(`Switched to ${mode === 'prowlarr' ? 'Prowlarr' : 'SlipStream'} mode`)
      onModeChange?.(mode)
    } catch {
      toast.error('Failed to change indexer mode')
    }
  }

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Loader2 className="text-muted-foreground size-6 animate-spin" />
        </CardContent>
      </Card>
    )
  }

  const effectiveMode = modeInfo?.effectiveMode ?? 'slipstream'

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Indexer Mode</CardTitle>
        <CardDescription>
          Choose how SlipStream manages indexers for searching releases
        </CardDescription>
      </CardHeader>
      <CardContent>
        <RadioGroup
          value={effectiveMode}
          onValueChange={(value) => handleModeChange(value as IndexerMode)}
          disabled={setModeMutation.isPending}
          className="space-y-3"
        >
          <div className="hover:bg-muted/50 has-[[data-state=checked]]:border-primary has-[[data-state=checked]]:bg-primary/5 flex items-start gap-3 rounded-lg border p-4 transition-colors">
            <RadioGroupItem value="slipstream" id="mode-slipstream" className="mt-1" />
            <Label htmlFor="mode-slipstream" className="flex-1 cursor-pointer space-y-1">
              <div className="flex items-center gap-2">
                <span className="font-medium">SlipStream Indexers</span>
                <Badge variant="outline" className="border-amber-500/50 text-xs text-amber-500">
                  Experimental
                </Badge>
              </div>
              <p className="text-muted-foreground text-sm">
                Use SlipStream&apos;s built-in Cardigann-based indexer management. Configure indexers
                directly within SlipStream.
              </p>
            </Label>
          </div>

          <div className="hover:bg-muted/50 has-[[data-state=checked]]:border-primary has-[[data-state=checked]]:bg-primary/5 flex items-start gap-3 rounded-lg border p-4 transition-colors">
            <RadioGroupItem value="prowlarr" id="mode-prowlarr" className="mt-1" />
            <Label htmlFor="mode-prowlarr" className="flex-1 cursor-pointer space-y-1">
              <div className="flex items-center gap-2">
                <span className="font-medium">Prowlarr</span>
                <ExternalLink className="text-muted-foreground size-3.5" />
              </div>
              <p className="text-muted-foreground text-sm">
                Connect to an external Prowlarr instance for centralized indexer management.
                Recommended if you&apos;re already using Prowlarr.
              </p>
            </Label>
          </div>
        </RadioGroup>

        {modeInfo?.devModeOverride ? (
          <div className="mt-4 flex items-center gap-2 text-sm text-amber-500">
            <AlertTriangle className="size-4" />
            <span>Developer mode is active - mode may be overridden</span>
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}

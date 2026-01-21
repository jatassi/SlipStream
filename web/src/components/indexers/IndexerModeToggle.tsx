import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { AlertTriangle, ExternalLink, Loader2 } from 'lucide-react'
import { useIndexerMode, useSetIndexerMode } from '@/hooks'
import { toast } from 'sonner'
import type { IndexerMode } from '@/types'

interface IndexerModeToggleProps {
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
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
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
          <div className="flex items-start gap-3 rounded-lg border p-4 hover:bg-muted/50 transition-colors has-[[data-state=checked]]:border-primary has-[[data-state=checked]]:bg-primary/5">
            <RadioGroupItem value="slipstream" id="mode-slipstream" className="mt-1" />
            <Label htmlFor="mode-slipstream" className="flex-1 cursor-pointer space-y-1">
              <div className="flex items-center gap-2">
                <span className="font-medium">SlipStream Indexers</span>
                <Badge variant="outline" className="text-amber-500 border-amber-500/50 text-xs">
                  Experimental
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground">
                Use SlipStream's built-in Cardigann-based indexer management. Configure indexers directly within SlipStream.
              </p>
            </Label>
          </div>

          <div className="flex items-start gap-3 rounded-lg border p-4 hover:bg-muted/50 transition-colors has-[[data-state=checked]]:border-primary has-[[data-state=checked]]:bg-primary/5">
            <RadioGroupItem value="prowlarr" id="mode-prowlarr" className="mt-1" />
            <Label htmlFor="mode-prowlarr" className="flex-1 cursor-pointer space-y-1">
              <div className="flex items-center gap-2">
                <span className="font-medium">Prowlarr</span>
                <ExternalLink className="size-3.5 text-muted-foreground" />
              </div>
              <p className="text-sm text-muted-foreground">
                Connect to an external Prowlarr instance for centralized indexer management. Recommended if you're already using Prowlarr.
              </p>
            </Label>
          </div>
        </RadioGroup>

        {modeInfo?.devModeOverride && (
          <div className="mt-4 flex items-center gap-2 text-sm text-amber-500">
            <AlertTriangle className="size-4" />
            <span>Developer mode is active - mode may be overridden</span>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

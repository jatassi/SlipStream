import { useState, useEffect } from 'react'
import { Loader2, TestTube, ArrowLeft, Globe, Lock, Unlock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { toast } from 'sonner'
import {
  useCreateIndexer,
  useUpdateIndexer,
  useTestIndexerConfig,
  useDefinitions,
  useDefinitionSchema,
} from '@/hooks'
import { DefinitionSearchTable } from './DefinitionSearchTable'
import { DynamicSettingsForm } from './DynamicSettingsForm'
import type { Indexer, CreateIndexerInput, DefinitionMetadata, Privacy, Protocol } from '@/types'

interface IndexerDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  indexer?: Indexer | null
}

type DialogStep = 'select' | 'configure'

const privacyIcons: Record<Privacy, React.ReactNode> = {
  public: <Globe className="size-4" />,
  'semi-private': <Unlock className="size-4" />,
  private: <Lock className="size-4" />,
}

const privacyColors: Record<Privacy, string> = {
  public: 'bg-green-500/10 text-green-500',
  'semi-private': 'bg-yellow-500/10 text-yellow-500',
  private: 'bg-red-500/10 text-red-500',
}

const protocolColors: Record<Protocol, string> = {
  torrent: 'bg-blue-500/10 text-blue-500',
  usenet: 'bg-purple-500/10 text-purple-500',
}

export function IndexerDialog({ open, onOpenChange, indexer }: IndexerDialogProps) {
  const [step, setStep] = useState<DialogStep>('select')
  const [selectedDefinition, setSelectedDefinition] = useState<DefinitionMetadata | null>(null)
  const [formData, setFormData] = useState<{
    name: string
    settings: Record<string, string>
    supportsMovies: boolean
    supportsTv: boolean
    priority: number
    enabled: boolean
  }>({
    name: '',
    settings: {},
    supportsMovies: true,
    supportsTv: true,
    priority: 50,
    enabled: true,
  })
  const [isTesting, setIsTesting] = useState(false)

  const { data: definitions = [], isLoading: isLoadingDefinitions } = useDefinitions()
  const { data: schema = [], isLoading: isLoadingSchema } = useDefinitionSchema(
    selectedDefinition?.id ?? ''
  )

  const createMutation = useCreateIndexer()
  const updateMutation = useUpdateIndexer()
  const testMutation = useTestIndexerConfig()

  const isEditing = !!indexer

  // Reset state when dialog opens/closes
  useEffect(() => {
    if (open) {
      if (indexer) {
        // Editing mode - go straight to configure with existing data
        const def = definitions.find((d) => d.id === indexer.definitionId)
        if (def) {
          setSelectedDefinition(def)
        } else {
          // Create a placeholder definition from indexer data
          setSelectedDefinition({
            id: indexer.definitionId,
            name: indexer.name,
            protocol: indexer.protocol,
            privacy: indexer.privacy,
          })
        }
        setFormData({
          name: indexer.name,
          settings: indexer.settings || {},
          supportsMovies: indexer.supportsMovies,
          supportsTv: indexer.supportsTv,
          priority: indexer.priority,
          enabled: indexer.enabled,
        })
        setStep('configure')
      } else {
        // Adding mode - start with selection
        setSelectedDefinition(null)
        setFormData({
          name: '',
          settings: {},
          supportsMovies: true,
          supportsTv: true,
          priority: 50,
          enabled: true,
        })
        setStep('select')
      }
    }
  }, [open, indexer, definitions])

  const handleDefinitionSelect = (def: DefinitionMetadata) => {
    setSelectedDefinition(def)
    setFormData((prev) => ({
      ...prev,
      name: def.name,
      settings: {},
    }))
    setStep('configure')
  }

  const handleBack = () => {
    setStep('select')
    setSelectedDefinition(null)
  }

  const handleTest = async () => {
    if (!selectedDefinition) return

    setIsTesting(true)
    try {
      const result = await testMutation.mutateAsync({
        definitionId: selectedDefinition.id,
        settings: formData.settings,
      })
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

  const handleSubmit = async () => {
    if (!formData.name.trim()) {
      toast.error('Name is required')
      return
    }
    if (!selectedDefinition) {
      toast.error('Please select an indexer definition')
      return
    }

    const input: CreateIndexerInput = {
      name: formData.name,
      definitionId: selectedDefinition.id,
      settings: formData.settings,
      supportsMovies: formData.supportsMovies,
      supportsTv: formData.supportsTv,
      priority: formData.priority,
      enabled: formData.enabled,
    }

    try {
      if (isEditing && indexer) {
        await updateMutation.mutateAsync({
          id: indexer.id,
          data: input,
        })
        toast.success('Indexer updated')
      } else {
        await createMutation.mutateAsync(input)
        toast.success('Indexer added')
      }
      onOpenChange(false)
    } catch {
      toast.error(isEditing ? 'Failed to update indexer' : 'Failed to add indexer')
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={step === 'select' ? 'sm:max-w-3xl h-[600px] flex flex-col overflow-hidden' : 'sm:max-w-2xl h-[80vh] flex flex-col overflow-hidden'}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {step === 'configure' && !isEditing && (
              <Button variant="ghost" size="icon" className="size-6" onClick={handleBack}>
                <ArrowLeft className="size-4" />
              </Button>
            )}
            {step === 'select' && 'Add Indexer'}
            {step === 'configure' && (isEditing ? 'Edit Indexer' : 'Configure Indexer')}
          </DialogTitle>
          <DialogDescription>
            {step === 'select' && 'Select an indexer from the list below.'}
            {step === 'configure' && 'Configure the indexer settings.'}
          </DialogDescription>
        </DialogHeader>

        {step === 'select' && (
          <div className="flex-1 min-h-0 overflow-hidden">
            <DefinitionSearchTable
              definitions={definitions}
              isLoading={isLoadingDefinitions}
              onSelect={handleDefinitionSelect}
            />
          </div>
        )}

        {step === 'configure' && selectedDefinition && (
          <ScrollArea className="flex-1 min-h-0">
            <div className="space-y-4 py-4 pr-4">
            {/* Definition Info Banner */}
            <div className="flex items-center gap-2 p-3 rounded-lg bg-muted/50">
              <div className="flex-1">
                <p className="font-medium">{selectedDefinition.name}</p>
                {selectedDefinition.description && (
                  <p className="text-sm text-muted-foreground">{selectedDefinition.description}</p>
                )}
              </div>
              <div className="flex gap-2">
                <Badge variant="secondary" className={protocolColors[selectedDefinition.protocol]}>
                  {selectedDefinition.protocol}
                </Badge>
                <Badge variant="secondary" className={privacyColors[selectedDefinition.privacy]}>
                  <span className="mr-1">{privacyIcons[selectedDefinition.privacy]}</span>
                  {selectedDefinition.privacy}
                </Badge>
              </div>
            </div>

            {/* Name */}
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                placeholder="My Indexer"
                value={formData.name}
                onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
              />
            </div>

            {/* Dynamic Settings */}
            {isLoadingSchema ? (
              <div className="flex items-center justify-center py-4">
                <Loader2 className="size-4 animate-spin mr-2" />
                Loading settings...
              </div>
            ) : (
              <DynamicSettingsForm
                settings={schema}
                values={formData.settings}
                onChange={(settings) => setFormData((prev) => ({ ...prev, settings }))}
              />
            )}

            {/* Supports Movies/TV */}
            <div className="grid grid-cols-2 gap-4">
              <div className="flex items-center justify-between">
                <Label htmlFor="supportsMovies">Movies</Label>
                <Switch
                  id="supportsMovies"
                  checked={formData.supportsMovies}
                  onCheckedChange={(checked) =>
                    setFormData((prev) => ({ ...prev, supportsMovies: checked }))
                  }
                />
              </div>
              <div className="flex items-center justify-between">
                <Label htmlFor="supportsTv">TV Shows</Label>
                <Switch
                  id="supportsTv"
                  checked={formData.supportsTv}
                  onCheckedChange={(checked) =>
                    setFormData((prev) => ({ ...prev, supportsTv: checked }))
                  }
                />
              </div>
            </div>

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
                  setFormData((prev) => ({ ...prev, priority: parseInt(e.target.value) || 50 }))
                }
              />
              <p className="text-xs text-muted-foreground">
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
          </ScrollArea>
        )}

        {step === 'configure' && (
          <DialogFooter className="flex-col gap-2 sm:flex-row">
            <Button variant="outline" onClick={handleTest} disabled={isTesting}>
              {isTesting ? (
                <Loader2 className="size-4 mr-2 animate-spin" />
              ) : (
                <TestTube className="size-4 mr-2" />
              )}
              Test
            </Button>
            <div className="flex gap-2 sm:ml-auto">
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button onClick={handleSubmit} disabled={isPending}>
                {isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
                {isEditing ? 'Save' : 'Add'}
              </Button>
            </div>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  )
}

import { useState } from 'react'

import { toast } from 'sonner'

import {
  useCreateIndexer,
  useDefinitions,
  useDefinitionSchema,
  useTestIndexerConfig,
  useUpdateIndexer,
} from '@/hooks'
import type { CreateIndexerInput, DefinitionMetadata, Indexer } from '@/types'

const EMPTY_DEFINITIONS: DefinitionMetadata[] = []

type DialogStep = 'select' | 'configure'

type FormData = {
  name: string
  settings: Record<string, string>
  supportsMovies: boolean
  supportsTv: boolean
  priority: number
  enabled: boolean
  autoSearchEnabled: boolean
  rssEnabled: boolean
}

const defaultFormData: FormData = {
  name: '',
  settings: {},
  supportsMovies: true,
  supportsTv: true,
  priority: 50,
  enabled: true,
  autoSearchEnabled: true,
  rssEnabled: true,
}

function createDefinitionFromIndexer(indexer: Indexer): DefinitionMetadata {
  return {
    id: indexer.definitionId,
    name: indexer.name,
    protocol: indexer.protocol,
    privacy: indexer.privacy,
  }
}

function createFormDataFromIndexer(indexer: Indexer): FormData {
  return {
    name: indexer.name,
    settings: indexer.settings ?? {},
    supportsMovies: indexer.supportsMovies,
    supportsTv: indexer.supportsTv,
    priority: indexer.priority,
    enabled: indexer.enabled,
    autoSearchEnabled: indexer.autoSearchEnabled,
    rssEnabled: indexer.rssEnabled,
  }
}

async function testConnection(
  selectedDefinition: DefinitionMetadata,
  settings: Record<string, string>,
  testMutation: ReturnType<typeof useTestIndexerConfig>,
) {
  const result = await testMutation.mutateAsync({
    definitionId: selectedDefinition.id,
    settings,
  })
  if (result.success) {
    toast.success(result.message || 'Connection successful')
  } else {
    toast.error(result.message || 'Connection failed')
  }
}

type SubmitParams = {
  formData: FormData
  selectedDefinition: DefinitionMetadata
  isEditing: boolean
  indexer: Indexer | null | undefined
  createMutation: ReturnType<typeof useCreateIndexer>
  updateMutation: ReturnType<typeof useUpdateIndexer>
}

async function submitForm(params: SubmitParams) {
  const { formData, selectedDefinition, isEditing, indexer, createMutation, updateMutation } =
    params

  const input: CreateIndexerInput = {
    name: formData.name,
    definitionId: selectedDefinition.id,
    settings: formData.settings,
    supportsMovies: formData.supportsMovies,
    supportsTv: formData.supportsTv,
    priority: formData.priority,
    enabled: formData.enabled,
    autoSearchEnabled: formData.autoSearchEnabled,
    rssEnabled: formData.rssEnabled,
  }

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
}

function useIndexerFormState(
  open: boolean,
  indexer: Indexer | null | undefined,
  definitions: DefinitionMetadata[],
) {
  const [step, setStep] = useState<DialogStep>('select')
  const [selectedDefinition, setSelectedDefinition] = useState<DefinitionMetadata | null>(null)
  const [formData, setFormData] = useState<FormData>(defaultFormData)
  const [isTesting, setIsTesting] = useState(false)
  const [prevOpen, setPrevOpen] = useState(open)
  const [prevIndexer, setPrevIndexer] = useState(indexer)
  const [prevDefs, setPrevDefs] = useState(definitions)

  if (open !== prevOpen || indexer !== prevIndexer || definitions !== prevDefs) {
    setPrevOpen(open); setPrevIndexer(indexer); setPrevDefs(definitions)
    if (open) {
      if (indexer) {
        const def = definitions.find((d) => d.id === indexer.definitionId)
        setSelectedDefinition(def ?? createDefinitionFromIndexer(indexer))
        setFormData(createFormDataFromIndexer(indexer))
        setStep('configure')
      } else {
        setSelectedDefinition(null)
        setFormData(defaultFormData)
        setStep('select')
      }
    }
  }

  return { step, setStep, selectedDefinition, setSelectedDefinition, formData, setFormData, isTesting, setIsTesting }
}

export function useIndexerDialog(
  open: boolean,
  indexer: Indexer | null | undefined,
  onOpenChange: (open: boolean) => void,
) {
  const { data: definitions = EMPTY_DEFINITIONS, isLoading: isLoadingDefinitions } = useDefinitions()
  const state = useIndexerFormState(open, indexer, definitions)
  const { data: schema = [], isLoading: isLoadingSchema } = useDefinitionSchema(
    state.selectedDefinition?.id ?? '',
  )
  const createMutation = useCreateIndexer()
  const updateMutation = useUpdateIndexer()
  const testMutation = useTestIndexerConfig()
  const isEditing = !!indexer

  return {
    ...state,
    definitions,
    isLoadingDefinitions,
    schema,
    isLoadingSchema,
    isEditing,
    isPending: createMutation.isPending || updateMutation.isPending,
    handleDefinitionSelect: (def: DefinitionMetadata) => {
      state.setSelectedDefinition(def)
      state.setFormData((prev) => ({ ...prev, name: def.name, settings: {} }))
      state.setStep('configure')
    },
    handleBack: () => { state.setStep('select'); state.setSelectedDefinition(null) },
    handleTest: async () => {
      if (!state.selectedDefinition) { return }
      state.setIsTesting(true)
      try { await testConnection(state.selectedDefinition, state.formData.settings, testMutation) }
      catch { toast.error('Failed to test connection') }
      finally { state.setIsTesting(false) }
    },
    handleSubmit: async () => {
      if (!state.formData.name.trim() || !state.selectedDefinition) {
        toast.error(state.formData.name.trim() ? 'Please select an indexer definition' : 'Name is required')
        return
      }
      try {
        await submitForm({ formData: state.formData, selectedDefinition: state.selectedDefinition, isEditing, indexer, createMutation, updateMutation })
        onOpenChange(false)
      } catch { toast.error(isEditing ? 'Failed to update indexer' : 'Failed to add indexer') }
    },
  }
}

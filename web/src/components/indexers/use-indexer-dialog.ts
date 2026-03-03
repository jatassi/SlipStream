import { useState } from 'react'
import { useForm } from 'react-hook-form'

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

export type FormData = {
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
    await updateMutation.mutateAsync({ id: indexer.id, data: input })
    toast.success('Indexer updated')
  } else {
    await createMutation.mutateAsync(input)
    toast.success('Indexer added')
  }
}

type HandlersState = ReturnType<typeof useIndexerFormState>
type HandlersDeps = {
  isEditing: boolean
  indexer: Indexer | null | undefined
  createMutation: ReturnType<typeof useCreateIndexer>
  updateMutation: ReturnType<typeof useUpdateIndexer>
  onOpenChange: (open: boolean) => void
}

function makeSubmitHandler(
  state: HandlersState,
  form: ReturnType<typeof useForm<FormData>>,
  deps: HandlersDeps,
) {
  const { isEditing, indexer, createMutation, updateMutation, onOpenChange } = deps
  return form.handleSubmit(async (formData) => {
    if (!state.selectedDefinition) {
      toast.error('Please select an indexer definition')
      return
    }
    try {
      await submitForm({
        formData,
        selectedDefinition: state.selectedDefinition,
        isEditing,
        indexer,
        createMutation,
        updateMutation,
      })
      onOpenChange(false)
    } catch {
      toast.error(isEditing ? 'Failed to update indexer' : 'Failed to add indexer')
    }
  })
}

type FormStateOptions = {
  open: boolean
  indexer: Indexer | null | undefined
  definitions: DefinitionMetadata[]
  form: ReturnType<typeof useForm<FormData>>
}

function useIndexerFormState({ open, indexer, definitions, form }: FormStateOptions) {
  const [step, setStep] = useState<DialogStep>('select')
  const [selectedDefinition, setSelectedDefinition] = useState<DefinitionMetadata | null>(null)
  const [isTesting, setIsTesting] = useState(false)
  const [prevOpen, setPrevOpen] = useState(open)
  const [prevIndexer, setPrevIndexer] = useState(indexer)
  const [prevDefs, setPrevDefs] = useState(definitions)

  if (open !== prevOpen || indexer !== prevIndexer || definitions !== prevDefs) {
    setPrevOpen(open)
    setPrevIndexer(indexer)
    setPrevDefs(definitions)
    if (open) {
      if (indexer) {
        const def = definitions.find((d) => d.id === indexer.definitionId)
        setSelectedDefinition(def ?? createDefinitionFromIndexer(indexer))
        form.reset(createFormDataFromIndexer(indexer))
        setStep('configure')
      } else {
        setSelectedDefinition(null)
        form.reset(defaultFormData)
        setStep('select')
      }
    }
  }

  return { step, setStep, selectedDefinition, setSelectedDefinition, isTesting, setIsTesting }
}

type BuildHandlersDeps = HandlersDeps & { testMutation: ReturnType<typeof useTestIndexerConfig> }

function buildHandlers(
  state: HandlersState,
  form: ReturnType<typeof useForm<FormData>>,
  deps: BuildHandlersDeps,
) {
  const { testMutation, ...submitDeps } = deps

  const handleDefinitionSelect = (def: DefinitionMetadata) => {
    state.setSelectedDefinition(def)
    form.reset({ ...defaultFormData, name: def.name, settings: {} })
    state.setStep('configure')
  }

  const handleBack = () => {
    state.setStep('select')
    state.setSelectedDefinition(null)
  }

  const handleTest = async () => {
    if (!state.selectedDefinition) {
      return
    }
    state.setIsTesting(true)
    try {
      await testConnection(state.selectedDefinition, form.getValues('settings'), testMutation)
    } catch {
      toast.error('Failed to test connection')
    } finally {
      state.setIsTesting(false)
    }
  }

  const handleSubmit = makeSubmitHandler(state, form, submitDeps)

  return { handleDefinitionSelect, handleBack, handleTest, handleSubmit }
}

export function useIndexerDialog(
  open: boolean,
  indexer: Indexer | null | undefined,
  onOpenChange: (open: boolean) => void,
) {
  const form = useForm<FormData>({ defaultValues: defaultFormData })
  const { data: definitions = EMPTY_DEFINITIONS, isLoading: isLoadingDefinitions } =
    useDefinitions()
  const state = useIndexerFormState({ open, indexer, definitions, form })
  const { data: schema = [], isLoading: isLoadingSchema } = useDefinitionSchema(
    state.selectedDefinition?.id ?? '',
  )
  const createMutation = useCreateIndexer()
  const updateMutation = useUpdateIndexer()
  const testMutation = useTestIndexerConfig()
  const isEditing = !!indexer

  const handlers = buildHandlers(state, form, {
    isEditing,
    indexer,
    createMutation,
    updateMutation,
    testMutation,
    onOpenChange,
  })

  return {
    ...state,
    ...handlers,
    form,
    definitions,
    isLoadingDefinitions,
    schema,
    isLoadingSchema,
    isEditing,
    isPending: createMutation.isPending || updateMutation.isPending,
  }
}

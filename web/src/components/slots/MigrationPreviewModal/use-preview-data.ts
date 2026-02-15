import { useEffect, useState } from 'react'

import { toast } from 'sonner'

import { useMigrationPreview } from '@/hooks'

import { generateDebugPreview } from './debug'
import type { MigrationPreview } from './types'

type TabType = 'movies' | 'tv'
type FilterType = 'all' | 'assigned' | 'conflicts' | 'nomatch'

function selectInitialTab(data: MigrationPreview): TabType {
  if (data.movies.length > 0) {
    return 'movies'
  }
  return data.tvShows.length > 0 ? 'tv' : 'movies'
}

type PreviewState = {
  preview: MigrationPreview | null
  activeTab: TabType
  isDebugData: boolean
  isLoadingDebugData: boolean
  filter: FilterType
}

const INITIAL_STATE: PreviewState = {
  preview: null,
  activeTab: 'movies',
  isDebugData: false,
  isLoadingDebugData: false,
  filter: 'all',
}

export function usePreviewData(open: boolean) {
  const previewMutation = useMigrationPreview()
  const [state, setState] = useState<PreviewState>(INITIAL_STATE)
  const [prevOpen, setPrevOpen] = useState(open)

  if (open !== prevOpen) {
    setPrevOpen(open)
    if (!open) {
      setState(INITIAL_STATE)
      previewMutation.reset()
    }
  }

  const { mutate, isPending, isError } = previewMutation
  useEffect(() => {
    if (!open || state.preview || isPending || isError) {
      return
    }
    mutate(undefined, {
      onSuccess: (data) => {
        setState((s) => ({ ...s, preview: data, activeTab: selectInitialTab(data) }))
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : 'Failed to generate preview')
      },
    })
  }, [open, state.preview, isPending, isError, mutate])

  const handleLoadDebugData = async () => {
    setState((s) => ({ ...s, isLoadingDebugData: true }))
    try {
      const debugPreview = await generateDebugPreview()
      setState((s) => ({
        ...s, preview: debugPreview, isDebugData: true, activeTab: 'movies', filter: 'all',
      }))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to generate debug data')
    } finally {
      setState((s) => ({ ...s, isLoadingDebugData: false }))
    }
  }

  return {
    ...state,
    setActiveTab: (tab: TabType) => setState((s) => ({ ...s, activeTab: tab })),
    setFilter: (f: FilterType) => setState((s) => ({ ...s, filter: f })),
    isLoading: isPending || state.isLoadingDebugData,
    handleLoadDebugData,
  }
}

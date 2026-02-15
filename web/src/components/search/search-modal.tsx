import { AlertCircle } from 'lucide-react'

import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'

import { SearchEmptyState, SearchErrorState, SearchLoadingState } from './search-empty-states'
import { SearchFooter } from './search-footer'
import { SearchInputBar } from './search-input-bar'
import type { SearchModalProps } from './search-modal-types'
import { SearchResultsTable } from './search-results-table'
import { useSearchModal } from './use-search-modal'

function SearchResultsArea(state: ReturnType<typeof useSearchModal>) {
  if (state.isLoading) {
    return <SearchLoadingState />
  }
  if (state.isError) {
    return <SearchErrorState error={state.error} onRetry={() => state.refetch()} />
  }
  if (state.releases.length === 0) {
    return <SearchEmptyState />
  }

  return (
    <SearchResultsTable
      releases={state.releases}
      sortColumn={state.sortColumn}
      sortDirection={state.sortDirection}
      grabbingGuid={state.grabbingGuid}
      hasTorrents={state.hasTorrents}
      hasSlotInfo={state.hasSlotInfo}
      onSort={state.handleSort}
      onGrab={state.handleGrab}
    />
  )
}

export function SearchModal(props: SearchModalProps) {
  const { open, onOpenChange } = props
  const state = useSearchModal(props)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[85vh] flex-col overflow-hidden sm:max-w-6xl">
        <DialogHeader>
          <DialogTitle>{state.title}</DialogTitle>
          <DialogDescription>
            Search indexers for releases and send to download client.
          </DialogDescription>
        </DialogHeader>

        <SearchInputBar
          query={state.query}
          isLoading={state.isLoading}
          onQueryChange={state.setQuery}
          onSearch={state.handleSearch}
        />

        {state.errors.length > 0 && (
          <Alert variant="destructive">
            <AlertCircle className="size-4" />
            <AlertDescription>
              {state.errors.length} indexer(s) returned errors. Some results may be missing.
            </AlertDescription>
          </Alert>
        )}

        <ScrollArea className="min-h-0 flex-1">
          <SearchResultsArea {...state} />
        </ScrollArea>

        {state.data ? (
          <SearchFooter
            total={state.data.total}
            indexersSearched={state.data.indexersSearched}
            errors={state.errors}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  )
}

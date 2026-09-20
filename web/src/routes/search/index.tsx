import { Screen } from '@/components/screen/screen'
import { SearchField } from '@/components/search/search-field'

import { AddNew, LibraryResults } from './search-groups'
import { useSearchPage } from './use-search-page'

function Hint({ children }: { children: React.ReactNode }) {
  return <p className="text-body text-muted-foreground px-screen pb-5">{children}</p>
}

function SearchBody({ state }: { state: ReturnType<typeof useSearchPage> }) {
  if (state.query.length === 0) {
    if (state.libraryLoading) {
      return null
    }
    return (
      <Hint>
        Search <span className="nums">{state.searchableCount}</span> titles across movies and series.
      </Hint>
    )
  }
  return (
    <>
      {state.results.length > 0 ? (
        <LibraryResults results={state.results} />
      ) : (
        <Hint>No library results for “{state.query}”.</Hint>
      )}
      {state.query.length >= 2 && <AddNew external={state.external} query={state.query} />}
    </>
  )
}

export function SearchPage() {
  const state = useSearchPage()

  return (
    <Screen title="Search">
      <div className="px-screen pb-5">
        <SearchField value={state.text} onChange={state.setText} placeholder="Search library" />
      </div>
      <SearchBody state={state} />
    </Screen>
  )
}

import { Rss } from 'lucide-react'

import { IconTile } from '@/components/grouped-list'
import {
  IndexerDialog,
  IndexerModeToggle,
  ProwlarrConfigForm,
  ProwlarrIndexerList,
} from '@/components/indexers'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import type { SettingsRowAction } from '@/components/settings/settings-item-row'
import { SettingsItemRow } from '@/components/settings/settings-item-row'
import { AddAction, SettingsList } from '@/components/settings/settings-list'
import type { Indexer } from '@/types'

import { useIndexersPage } from './use-indexers-page'

type PageState = ReturnType<typeof useIndexersPage>

export function IndexersPage() {
  const page = useIndexersPage()
  const back = usePushBack()

  return (
    <Screen
      title="Indexers"
      back={back}
      trailing={
        page.isProwlarrMode ? undefined : (
          <AddAction label="Add indexer" onClick={page.handleAdd} />
        )
      }
    >
      <div className="px-screen mb-7">
        <IndexerModeToggle />
      </div>
      {page.isProwlarrMode ? <ProwlarrMode /> : <IndexerList page={page} />}
      <IndexerDialog
        open={page.dialogOpen}
        onOpenChange={page.setDialogOpen}
        indexer={page.editingIndexer}
      />
    </Screen>
  )
}

function ProwlarrMode() {
  return (
    <div className="px-screen space-y-6">
      <ProwlarrConfigForm />
      <ProwlarrIndexerList />
    </div>
  )
}

function IndexerList({ page }: { page: PageState }) {
  const { data: indexers, isLoading, isError, refetch } = page.query

  return (
    <SettingsList
      state={{
        isLoading: isLoading || page.modeLoading,
        isError,
        isEmpty: !indexers?.length,
        refetch,
      }}
      empty="No indexers yet"
    >
      {indexers?.map((indexer) => (
        <IndexerRow key={indexer.id} indexer={indexer} page={page} />
      ))}
    </SettingsList>
  )
}

function IndexerRow({ indexer, page }: { indexer: Indexer; page: PageState }) {
  const actions: SettingsRowAction[] = [
    { label: 'Test', onClick: () => page.handleTest(indexer.id) },
    {
      label: 'Delete',
      destructive: true,
      onClick: () => page.handleDelete(indexer.id),
      confirm: {
        title: 'Delete indexer',
        description: `Are you sure you want to delete "${indexer.name}"?`,
      },
    },
  ]

  return (
    <SettingsItemRow
      leading={
        <IconTile className="bg-tv-600">
          <Rss />
        </IconTile>
      }
      title={indexer.name}
      subtitle={indexerSubtitle(indexer)}
      onOpen={() => page.handleEdit(indexer)}
      openLabel={`Edit ${indexer.name}`}
      toggle={{
        label: `${indexer.name} enabled`,
        checked: indexer.enabled,
        onCheckedChange: (checked) => page.handleToggleEnabled(indexer.id, checked),
      }}
      actions={actions}
    />
  )
}

function mediaSupport(indexer: Indexer): string {
  if (indexer.supportsMovies && indexer.supportsTv) {
    return 'Movies / TV'
  }
  if (indexer.supportsMovies) {
    return 'Movies'
  }
  return indexer.supportsTv ? 'TV' : 'No media types'
}

function indexerSubtitle(indexer: Indexer): string {
  const parts = [indexer.protocol, mediaSupport(indexer), `Priority ${indexer.priority}`]
  if (!indexer.autoSearchEnabled) {
    parts.push('Manual search only')
  }
  if (!indexer.rssEnabled) {
    parts.push('No RSS')
  }
  return parts.join(' · ')
}

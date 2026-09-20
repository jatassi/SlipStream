import { Film, Tv } from 'lucide-react'

import { IconTile } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import type { SettingsRowAction } from '@/components/settings/settings-item-row'
import { SettingsItemRow } from '@/components/settings/settings-item-row'
import { AddAction, SettingsList } from '@/components/settings/settings-list'
import { formatBytes } from '@/lib/formatters'
import type { RootFolder } from '@/types'

import { AddRootFolderDialog } from './root-folder-dialog'
import { useRootFoldersPage } from './use-root-folders-page'

type PageState = ReturnType<typeof useRootFoldersPage>

export function RootFoldersPage() {
  const page = useRootFoldersPage()
  const back = usePushBack()
  const { data: folders, isLoading, isError, refetch } = page.query

  return (
    <Screen
      title="Root Folders"
      back={back}
      trailing={<AddAction label="Add root folder" onClick={() => page.setShowAddDialog(true)} />}
    >
      <SettingsList
        state={{ isLoading, isError, isEmpty: !folders?.length, refetch }}
        empty="No root folders yet"
      >
        {folders?.map((folder) => (
          <FolderRow key={folder.id} folder={folder} page={page} />
        ))}
      </SettingsList>
      <AddRootFolderDialog page={page} />
    </Screen>
  )
}

function folderActions(folder: RootFolder, page: PageState): SettingsRowAction[] {
  const defaultAction: SettingsRowAction = folder.isDefault
    ? { label: 'Clear default', onClick: () => page.handleClearDefault(folder.mediaType) }
    : {
        label: 'Set as default',
        onClick: () => page.handleSetDefault(folder.id, folder.mediaType),
      }
  return [
    defaultAction,
    {
      label: 'Delete',
      destructive: true,
      onClick: () => page.handleDelete(folder.id),
      confirm: {
        title: 'Delete root folder',
        description: `Are you sure you want to delete "${folder.name}" (${folder.path})?`,
      },
    },
  ]
}

function FolderRow({ folder, page }: { folder: RootFolder; page: PageState }) {
  const isMovie = folder.mediaType === 'movie'

  return (
    <SettingsItemRow
      leading={
        <IconTile className={isMovie ? 'bg-movie-600' : 'bg-tv-600'}>
          {isMovie ? <Film /> : <Tv />}
        </IconTile>
      }
      title={folder.name}
      subtitle={<span className="font-mono">{folder.path}</span>}
      detail={folderDetail(folder)}
      actions={folderActions(folder, page)}
    />
  )
}

function folderDetail(folder: RootFolder): string | undefined {
  const free = folder.freeSpace > 0 ? `${formatBytes(folder.freeSpace)} free` : ''
  if (folder.isDefault) {
    return free === '' ? 'Default' : `Default · ${free}`
  }
  return free === '' ? undefined : free
}

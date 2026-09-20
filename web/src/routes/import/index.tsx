import { Group, Row } from '@/components/grouped-list'
import { Screen } from '@/components/screen/screen'
import type { ScannedFile } from '@/types'

import { BrowserSkeleton, FileGroup, FolderGroup } from './browser-groups'
import { EditMatchDialog } from './edit-match-dialog'
import { importableFile } from './match-label'
import { PendingImportsGroup } from './pending-imports-group'
import { useImportActions } from './use-import-actions'
import { useImportBrowser } from './use-import-browser'
import { useListScroll } from './use-list-scroll'

const IMPORT_ALL =
  'press-dim min-h-tap text-title text-primary focus-visible:ring-ring px-2 font-semibold outline-none focus-visible:ring-[3px]'

type Browser = ReturnType<typeof useImportBrowser>
type Actions = ReturnType<typeof useImportActions>

function ImportAllAction({
  files,
  disabled,
  onImportAll,
}: {
  files: ScannedFile[]
  disabled: boolean
  onImportAll: (files: ScannedFile[]) => void
}) {
  if (files.length < 2) {
    return undefined
  }
  return (
    <button
      type="button"
      disabled={disabled}
      className={IMPORT_ALL}
      onClick={() => {
        onImportAll(files)
      }}
    >
      Import All
    </button>
  )
}

function EmptyFolder() {
  return (
    <Group>
      <Row title="Nothing here" subtitle="This folder holds no folders or video files" />
    </Group>
  )
}

function BrowserBody({
  browser,
  actions,
  files,
}: {
  browser: Browser
  actions: Actions
  files: Browser['files']
}) {
  if (browser.isLoading) {
    return <BrowserSkeleton />
  }
  if (browser.folders.length === 0 && files.length === 0) {
    return <EmptyFolder />
  }
  return (
    <>
      <FolderGroup folders={browser.folders} onOpen={browser.open} />
      <FileGroup
        files={files}
        matches={browser.matches}
        matching={browser.matching}
        isImporting={actions.isImporting}
        onImport={(scanned) => {
          void actions.importFile(scanned)
        }}
        onEdit={actions.setEditing}
      />
    </>
  )
}

function matchedFiles(browser: Browser, files: Browser['files']): ScannedFile[] {
  return files
    .map((file) => importableFile(browser.matches.get(file.path)))
    .filter((scanned) => scanned !== undefined)
}

export function ManualImportPage() {
  const browser = useImportBrowser()
  const actions = useImportActions()
  const scrollRef = useListScroll(browser.path, !browser.isLoading)
  const files = browser.files.filter((file) => !actions.imported.has(file.path))

  return (
    <Screen
      title={browser.title}
      back={browser.back}
      scrollRef={scrollRef}
      trailing={
        <ImportAllAction
          files={matchedFiles(browser, files)}
          disabled={actions.isImporting}
          onImportAll={(all) => {
            void actions.importAll(all)
          }}
        />
      }
    >
      <BrowserBody browser={browser} actions={actions} files={files} />
      {browser.path === '' && <PendingImportsGroup />}
      <EditMatchDialog
        key={actions.editing?.path}
        file={actions.editing}
        open={actions.editing !== null}
        onClose={() => {
          actions.setEditing(null)
        }}
        onConfirm={(file, match) => {
          void actions.confirmMatch(file, match)
        }}
      />
    </Screen>
  )
}

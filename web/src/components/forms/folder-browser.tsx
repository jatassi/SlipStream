import { Loader2 } from 'lucide-react'

import { FormActions, SheetPresenter } from '@/components/presenter'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'

import {
  Breadcrumbs,
  DrivesList,
  EmptyMessage,
  EntriesList,
  ParentButton,
  PathInput,
  SelectedPath,
} from './folder-browser-parts'
import { useFolderBrowser } from './use-folder-browser'

type FolderBrowserProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialPath?: string
  onSelect: (path: string) => void
  fileExtensions?: string[]
  /** Set when the browser opens from inside another presented form. */
  nested?: boolean
}

export function FolderBrowser({
  open,
  onOpenChange,
  initialPath = '',
  onSelect,
  fileExtensions,
  nested,
}: FolderBrowserProps) {
  const s = useFolderBrowser({ initialPath, open, onSelect, onOpenChange, fileExtensions })

  return (
    <SheetPresenter
      open={open}
      onOpenChange={onOpenChange}
      nested={nested}
      title={s.showFiles ? 'Browse Files' : 'Browse Folders'}
      wideClassName="sm:max-w-2xl"
      footer={
        <FormActions
          onCancel={() => onOpenChange(false)}
          confirmLabel={s.showFiles ? 'Select File' : 'Select Folder'}
          onConfirm={s.handleSelect}
          confirmDisabled={!s.selectedPath}
        />
      }
    >
      <div className="space-y-3 py-2">
        <PathInput
          inputPath={s.inputPath}
          setInputPath={s.setInputPath}
          hasDrives={!!s.data?.drives}
          onSubmit={s.handleInputSubmit}
        />
        {s.breadcrumbs.length > 0 && (
          <Breadcrumbs breadcrumbs={s.breadcrumbs} onNavigate={s.handleNavigate} />
        )}
        <BrowserContent
          isLoading={s.isLoading}
          error={s.error}
          data={s.data}
          onNavigate={s.handleNavigate}
          onRetry={s.refetch}
          onFileSelect={s.handleFileSelect}
          selectedFile={s.selectedFile}
        />
        {s.selectedPath ? <SelectedPath path={s.selectedPath} /> : null}
      </div>
    </SheetPresenter>
  )
}

type BrowserData = {
  drives?: { letter: string; label?: string; freeSpace?: number }[]
  parent?: string
  path?: string
  entries?: { path: string; name: string; isDir: boolean }[]
}

function BrowserContent({
  isLoading,
  error,
  data,
  onNavigate,
  onRetry,
  onFileSelect,
  selectedFile,
}: {
  isLoading: boolean
  error: Error | null
  data: BrowserData | undefined
  onNavigate: (path: string) => void
  onRetry: () => void
  onFileSelect?: (path: string) => void
  selectedFile: string
}) {
  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center rounded-lg border">
        <Loader2 className="text-muted-foreground size-6 animate-spin" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-64 flex-col items-center justify-center rounded-lg border p-4 text-center">
        <p className="text-destructive mb-2">Failed to load directory</p>
        <p className="text-muted-foreground mb-4 text-sm">
          {error instanceof Error ? error.message : 'Unknown error'}
        </p>
        <Button variant="outline" size="sm" onClick={onRetry}>
          Retry
        </Button>
      </div>
    )
  }

  return (
    <div className="rounded-lg border">
      <ScrollArea className="h-64">
        <DirectoryContent
          data={data}
          onNavigate={onNavigate}
          onFileSelect={onFileSelect}
          selectedFile={selectedFile}
        />
      </ScrollArea>
    </div>
  )
}

function DirectoryContent({
  data,
  onNavigate,
  onFileSelect,
  selectedFile,
}: {
  data: BrowserData | undefined
  onNavigate: (path: string) => void
  onFileSelect?: (path: string) => void
  selectedFile: string
}) {
  if (!data) {
    return null
  }

  if (data.drives && data.drives.length > 0) {
    return <DrivesList drives={data.drives} onNavigate={onNavigate} />
  }

  const entries = data.entries ?? []
  if (!data.parent && entries.length === 0) {
    return data.path && <EmptyMessage />
  }

  return (
    <>
      {data.parent ? <ParentButton parent={data.parent} onNavigate={onNavigate} /> : null}
      {entries.length > 0 && (
        <EntriesList
          entries={entries}
          onNavigate={onNavigate}
          onFileSelect={onFileSelect}
          selectedFile={selectedFile}
        />
      )}
    </>
  )
}

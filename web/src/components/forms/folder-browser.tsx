import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
}

export function FolderBrowser({
  open,
  onOpenChange,
  initialPath = '',
  onSelect,
  fileExtensions,
}: FolderBrowserProps) {
  const s = useFolderBrowser({ initialPath, open, onSelect, onOpenChange, fileExtensions })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>{s.showFiles ? 'Browse Files' : 'Browse Folders'}</DialogTitle>
        </DialogHeader>

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

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={s.handleSelect} disabled={!s.selectedPath}>
            {s.showFiles ? 'Select File' : 'Select Folder'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
    return data.path ? <EmptyMessage /> : null
  }

  return (
    <>
      {data.parent ? <ParentButton parent={data.parent} onNavigate={onNavigate} /> : null}
      {entries.length > 0 ? (
        <EntriesList
          entries={entries}
          onNavigate={onNavigate}
          onFileSelect={onFileSelect}
          selectedFile={selectedFile}
        />
      ) : null}
    </>
  )
}

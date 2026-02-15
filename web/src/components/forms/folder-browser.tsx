import { useState } from 'react'

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
import { useBrowseDirectory } from '@/hooks'

import {
  Breadcrumbs,
  DrivesList,
  EmptyMessage,
  EntriesList,
  ParentButton,
  PathInput,
  SelectedPath,
} from './folder-browser-parts'

const getBreadcrumbs = (path: string) => {
  if (!path) {
    return []
  }

  // Handle Windows paths
  const isWindows = /^[A-Za-z]:/.test(path)
  const parts = path.split(/[/\\]/).filter(Boolean)

  const breadcrumbs: { label: string; path: string }[] = []
  let accumulated = isWindows ? '' : '/'

  for (const part of parts) {
    if (isWindows) {
      accumulated = accumulated ? `${accumulated}\\${part}` : part
    } else {
      accumulated = `${accumulated}${accumulated === '/' ? '' : '/'}${part}`
    }

    // For Windows, add : after drive letter
    const displayPath = isWindows && breadcrumbs.length === 0 ? `${part}:\\` : accumulated

    breadcrumbs.push({
      label: part,
      path: displayPath,
    })
  }

  return breadcrumbs
}

type FolderBrowserProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialPath?: string
  onSelect: (path: string) => void
}

export function FolderBrowser({ open, onOpenChange, initialPath = '', onSelect }: FolderBrowserProps) {
  const [currentPath, setCurrentPath] = useState(initialPath)
  const [inputPath, setInputPath] = useState(initialPath)
  const { data, isLoading, error, refetch } = useBrowseDirectory(currentPath || undefined, open)

  const handleNavigate = (path: string) => {
    setCurrentPath(path)
    setInputPath(path)
  }

  const handleInputSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setCurrentPath(inputPath)
  }

  const handleSelect = () => {
    onSelect(currentPath || inputPath)
    onOpenChange(false)
  }

  const breadcrumbs = getBreadcrumbs(data?.path ?? currentPath)
  const selectedPath = currentPath || inputPath

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>Browse Folders</DialogTitle>
        </DialogHeader>

        <BrowserMain
          inputPath={inputPath}
          setInputPath={setInputPath}
          hasDrives={!!data?.drives}
          handleInputSubmit={handleInputSubmit}
          breadcrumbs={breadcrumbs}
          handleNavigate={handleNavigate}
          isLoading={isLoading}
          error={error}
          data={data}
          refetch={refetch}
          selectedPath={selectedPath}
        />

        <BrowserFooter
          onCancel={() => onOpenChange(false)}
          onSelect={handleSelect}
          hasSelection={!!selectedPath}
        />
      </DialogContent>
    </Dialog>
  )
}

function BrowserMain({
  inputPath,
  setInputPath,
  hasDrives,
  handleInputSubmit,
  breadcrumbs,
  handleNavigate,
  isLoading,
  error,
  data,
  refetch,
  selectedPath,
}: {
  inputPath: string
  setInputPath: (path: string) => void
  hasDrives: boolean
  handleInputSubmit: (e: React.FormEvent) => void
  breadcrumbs: { label: string; path: string }[]
  handleNavigate: (path: string) => void
  isLoading: boolean
  error: Error | null
  data: BrowserData | undefined
  refetch: () => void
  selectedPath: string
}) {
  return (
    <>
      <PathInput
        inputPath={inputPath}
        setInputPath={setInputPath}
        hasDrives={hasDrives}
        onSubmit={handleInputSubmit}
      />

      {breadcrumbs.length > 0 && (
        <Breadcrumbs breadcrumbs={breadcrumbs} onNavigate={handleNavigate} />
      )}

      <BrowserContent
        isLoading={isLoading}
        error={error}
        data={data}
        onNavigate={handleNavigate}
        onRetry={refetch}
      />

      {selectedPath ? <SelectedPath path={selectedPath} /> : null}
    </>
  )
}

function BrowserFooter({
  onCancel,
  onSelect,
  hasSelection,
}: {
  onCancel: () => void
  onSelect: () => void
  hasSelection: boolean
}) {
  return (
    <DialogFooter>
      <Button variant="outline" onClick={onCancel}>
        Cancel
      </Button>
      <Button onClick={onSelect} disabled={!hasSelection}>
        Select Folder
      </Button>
    </DialogFooter>
  )
}

type BrowserData = {
  drives?: { letter: string; label?: string; freeSpace?: number }[]
  parent?: string
  path?: string
  entries?: { path: string; name: string }[]
}

function BrowserContent({
  isLoading,
  error,
  data,
  onNavigate,
  onRetry,
}: {
  isLoading: boolean
  error: Error | null
  data: BrowserData | undefined
  onNavigate: (path: string) => void
  onRetry: () => void
}) {
  if (isLoading) {
    return <LoadingView />
  }
  if (error) {
    return <ErrorView error={error} onRetry={onRetry} />
  }
  return <ContentView data={data} onNavigate={onNavigate} />
}

function LoadingView() {
  return (
    <div className="flex h-64 items-center justify-center rounded-lg border">
      <Loader2 className="text-muted-foreground size-6 animate-spin" />
    </div>
  )
}

function ErrorView({ error, onRetry }: { error: Error; onRetry: () => void }) {
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

function ContentView({ data, onNavigate }: { data: BrowserData | undefined; onNavigate: (path: string) => void }) {
  return (
    <div className="rounded-lg border">
      <ScrollArea className="h-64">
        <ContentItems data={data} onNavigate={onNavigate} />
      </ScrollArea>
    </div>
  )
}

function ContentItems({ data, onNavigate }: { data: BrowserData | undefined; onNavigate: (path: string) => void }) {
  if (!data) {
    return null
  }
  if (data.drives && data.drives.length > 0) {
    return <DrivesList drives={data.drives} onNavigate={onNavigate} />
  }
  return <DirectoryOrEmpty data={data} onNavigate={onNavigate} />
}

function DirectoryOrEmpty({ data, onNavigate }: { data: BrowserData; onNavigate: (path: string) => void }) {
  const hasContent = data.parent ?? (data.entries && data.entries.length > 0)
  if (hasContent) {
    return <DirectoryContents data={data} onNavigate={onNavigate} />
  }
  return data.path && !data.drives ? <EmptyMessage /> : null
}

function DirectoryContents({ data, onNavigate }: { data: BrowserData; onNavigate: (path: string) => void }) {
  return (
    <>
      {data.parent ? <ParentButton parent={data.parent} onNavigate={onNavigate} /> : null}
      {data.entries && data.entries.length > 0 ? <EntriesList entries={data.entries} onNavigate={onNavigate} /> : null}
    </>
  )
}

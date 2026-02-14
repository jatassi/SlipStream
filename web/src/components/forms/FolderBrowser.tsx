import { useState } from 'react'

import { ChevronRight, Folder, FolderUp, HardDrive, Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useBrowseDirectory } from '@/hooks'
import { formatBytes } from '@/lib/formatters'
import { cn } from '@/lib/utils'

type FolderBrowserProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialPath?: string
  onSelect: (path: string) => void
}

export function FolderBrowser({
  open,
  onOpenChange,
  initialPath = '',
  onSelect,
}: FolderBrowserProps) {
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

  // Build breadcrumb parts from path
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
      accumulated = isWindows
        ? accumulated
          ? `${accumulated}\\${part}`
          : part
        : `${accumulated}${accumulated === '/' ? '' : '/'}${part}`

      // For Windows, add : after drive letter
      const displayPath = isWindows && breadcrumbs.length === 0 ? `${part}:\\` : accumulated

      breadcrumbs.push({
        label: part,
        path: displayPath,
      })
    }

    return breadcrumbs
  }

  const breadcrumbs = getBreadcrumbs(data?.path || currentPath)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>Browse Folders</DialogTitle>
        </DialogHeader>

        {/* Path input */}
        <form onSubmit={handleInputSubmit} className="flex gap-2">
          <Input
            value={inputPath}
            onChange={(e) => setInputPath(e.target.value)}
            placeholder={data?.drives ? 'Select a drive or enter path...' : '/path/to/folder'}
            className="font-mono text-sm"
          />
          <Button type="submit" variant="outline" size="sm">
            Go
          </Button>
        </form>

        {/* Breadcrumb navigation */}
        {breadcrumbs.length > 0 && (
          <div className="flex items-center gap-1 overflow-x-auto pb-1 text-sm">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2"
              onClick={() => handleNavigate('')}
            >
              <HardDrive className="size-4" />
            </Button>
            {breadcrumbs.map((crumb, index) => (
              <div key={crumb.path} className="flex items-center">
                <ChevronRight className="text-muted-foreground size-4" />
                <Button
                  variant="ghost"
                  size="sm"
                  className={cn('h-7 px-2', index === breadcrumbs.length - 1 && 'font-medium')}
                  onClick={() => handleNavigate(crumb.path)}
                >
                  {crumb.label}
                </Button>
              </div>
            ))}
          </div>
        )}

        {/* Content area */}
        <div className="rounded-lg border">
          {isLoading ? (
            <div className="flex h-64 items-center justify-center">
              <Loader2 className="text-muted-foreground size-6 animate-spin" />
            </div>
          ) : error ? (
            <div className="flex h-64 flex-col items-center justify-center p-4 text-center">
              <p className="text-destructive mb-2">Failed to load directory</p>
              <p className="text-muted-foreground mb-4 text-sm">
                {error instanceof Error ? error.message : 'Unknown error'}
              </p>
              <Button variant="outline" size="sm" onClick={() => refetch()}>
                Retry
              </Button>
            </div>
          ) : (
            <ScrollArea className="h-64">
              {/* Drives (Windows root) */}
              {data?.drives && data.drives.length > 0 ? (
                <div className="p-2">
                  <p className="text-muted-foreground mb-2 px-2 text-xs font-medium">Drives</p>
                  {data.drives.map((drive) => (
                    <button
                      key={drive.letter}
                      className="hover:bg-accent flex w-full items-center gap-3 rounded-md px-3 py-2 text-left"
                      onClick={() => handleNavigate(`${drive.letter}\\`)}
                    >
                      <HardDrive className="text-muted-foreground size-5" />
                      <div className="flex-1">
                        <span className="font-medium">{drive.letter}</span>
                        {drive.label ? (
                          <span className="text-muted-foreground ml-2">{drive.label}</span>
                        ) : null}
                      </div>
                      {drive.freeSpace !== undefined && drive.freeSpace > 0 && (
                        <span className="text-muted-foreground text-xs">
                          {formatBytes(drive.freeSpace)} free
                        </span>
                      )}
                    </button>
                  ))}
                </div>
              ) : null}

              {/* Parent directory */}
              {data?.parent ? (
                <button
                  className="hover:bg-accent flex w-full items-center gap-3 border-b px-3 py-2 text-left"
                  onClick={() => handleNavigate(data.parent!)}
                >
                  <FolderUp className="text-muted-foreground size-5" />
                  <span className="text-muted-foreground">..</span>
                </button>
              ) : null}

              {/* Directory entries */}
              {data?.entries && data.entries.length > 0 ? (
                <div className="p-1">
                  {data.entries.map((entry) => (
                    <button
                      key={entry.path}
                      className="hover:bg-accent flex w-full items-center gap-3 rounded-md px-3 py-2 text-left"
                      onClick={() => handleNavigate(entry.path)}
                    >
                      <Folder className="size-5 text-blue-500" />
                      <span className="truncate">{entry.name}</span>
                    </button>
                  ))}
                </div>
              ) : data?.path && !data?.drives ? (
                <div className="text-muted-foreground flex h-32 items-center justify-center">
                  No subdirectories
                </div>
              ) : null}
            </ScrollArea>
          )}
        </div>

        {/* Selected path display */}
        {currentPath || inputPath ? (
          <div className="text-sm">
            <span className="text-muted-foreground">Selected: </span>
            <span className="font-mono">{currentPath || inputPath}</span>
          </div>
        ) : null}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSelect} disabled={!currentPath && !inputPath}>
            Select Folder
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

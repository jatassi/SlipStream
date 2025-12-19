import { useState } from 'react'
import { Plus, Trash2, FolderOpen, Film, Tv, HardDrive, FolderSearch } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { FolderBrowser } from '@/components/forms/FolderBrowser'
import { useRootFolders, useCreateRootFolder, useDeleteRootFolder } from '@/hooks'
import { formatBytes } from '@/lib/formatters'
import { toast } from 'sonner'

export function RootFoldersPage() {
  const [showAddDialog, setShowAddDialog] = useState(false)
  const [showBrowser, setShowBrowser] = useState(false)
  const [newPath, setNewPath] = useState('')
  const [newMediaType, setNewMediaType] = useState<'movie' | 'tv'>('movie')

  const { data: folders, isLoading, isError, refetch } = useRootFolders()
  const createMutation = useCreateRootFolder()
  const deleteMutation = useDeleteRootFolder()

  const handleAdd = async () => {
    if (!newPath.trim()) {
      toast.error('Please enter a path')
      return
    }

    try {
      await createMutation.mutateAsync({
        path: newPath,
        mediaType: newMediaType,
      })
      toast.success('Root folder added')
      setShowAddDialog(false)
      setNewPath('')
    } catch {
      toast.error('Failed to add root folder')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
      toast.success('Root folder deleted')
    } catch {
      toast.error('Failed to delete root folder')
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Root Folders" />
        <LoadingState variant="list" count={3} />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Root Folders" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Root Folders"
        description="Manage media library storage locations"
        breadcrumbs={[
          { label: 'Settings', href: '/settings' },
          { label: 'Root Folders' },
        ]}
        actions={
          <Button onClick={() => setShowAddDialog(true)}>
            <Plus className="size-4 mr-2" />
            Add Folder
          </Button>
        }
      />

      {!folders?.length ? (
        <EmptyState
          icon={<FolderOpen className="size-8" />}
          title="No root folders"
          description="Add a root folder to store your media"
          action={{ label: 'Add Folder', onClick: () => setShowAddDialog(true) }}
        />
      ) : (
        <div className="space-y-4">
          {folders.map((folder) => (
            <Card key={folder.id}>
              <CardHeader className="flex flex-row items-center justify-between py-4">
                <div className="flex items-center gap-4">
                  <div className="flex size-10 items-center justify-center rounded-lg bg-muted">
                    {folder.mediaType === 'movie' ? (
                      <Film className="size-5" />
                    ) : (
                      <Tv className="size-5" />
                    )}
                  </div>
                  <div>
                    <CardTitle className="text-base">{folder.name}</CardTitle>
                    <CardDescription className="font-mono text-xs">
                      {folder.path}
                    </CardDescription>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div className="text-right">
                    <Badge variant="secondary">{folder.mediaType}</Badge>
                    {folder.freeSpace > 0 && (
                      <p className="text-xs text-muted-foreground mt-1">
                        <HardDrive className="size-3 inline mr-1" />
                        {formatBytes(folder.freeSpace)} free
                      </p>
                    )}
                  </div>
                  <ConfirmDialog
                    trigger={
                      <Button variant="ghost" size="icon">
                        <Trash2 className="size-4" />
                      </Button>
                    }
                    title="Delete root folder"
                    description={`Are you sure you want to delete "${folder.path}"?`}
                    confirmLabel="Delete"
                    variant="destructive"
                    onConfirm={() => handleDelete(folder.id)}
                  />
                </div>
              </CardHeader>
            </Card>
          ))}
        </div>
      )}

      {/* Add dialog */}
      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Root Folder</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="path">Path</Label>
              <div className="flex gap-2">
                <Input
                  id="path"
                  placeholder="/path/to/media or C:\path\to\media"
                  value={newPath}
                  onChange={(e) => setNewPath(e.target.value)}
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  onClick={() => setShowBrowser(true)}
                  title="Browse folders"
                >
                  <FolderSearch className="size-4" />
                </Button>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="mediaType">Media Type</Label>
              <Select value={newMediaType} onValueChange={(v) => v && setNewMediaType(v as 'movie' | 'tv')}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="movie">Movies</SelectItem>
                  <SelectItem value="tv">TV Shows</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddDialog(false)}>
              Cancel
            </Button>
            <Button onClick={handleAdd} disabled={createMutation.isPending}>
              Add
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Folder browser dialog */}
      <FolderBrowser
        open={showBrowser}
        onOpenChange={setShowBrowser}
        initialPath={newPath}
        onSelect={(path) => setNewPath(path)}
      />
    </div>
  )
}

import { useState } from 'react'

import { Check, Film, FolderOpen, FolderSearch, HardDrive, Trash2, Tv, X } from 'lucide-react'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { FolderBrowser } from '@/components/forms/folder-browser'
import { ListSection } from '@/components/settings/list-section'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCreateRootFolder, useDeleteRootFolder, useRootFolders } from '@/hooks'
import { useClearDefault, useSetDefault } from '@/hooks/use-defaults'
import { formatBytes } from '@/lib/formatters'
import { withToast } from '@/lib/with-toast'
import type { MediaType, RootFolder } from '@/types'

type FolderActions = {
  onDelete: (id: number) => void
  onSetDefault: (id: number, mediaType: MediaType) => void
  onClearDefault: (mediaType: MediaType) => void
}

function FolderInfo({ folder }: { folder: RootFolder }) {
  return (
    <div className="flex items-center gap-4">
      <div className="bg-muted flex size-10 items-center justify-center rounded-lg">
        {folder.mediaType === 'movie' ? <Film className="size-5" /> : <Tv className="size-5" />}
      </div>
      <div>
        <CardTitle className="text-base">{folder.name}</CardTitle>
        <CardDescription className="font-mono text-xs">{folder.path}</CardDescription>
      </div>
    </div>
  )
}

function FolderBadges({ folder }: { folder: RootFolder }) {
  return (
    <div className="text-right">
      <div className="mb-1 flex items-center gap-2">
        <Badge variant="secondary">{folder.mediaType}</Badge>
        {folder.isDefault ? (
          <Badge variant="default" className="bg-green-500 hover:bg-green-600">
            <Check className="mr-1 size-3" />
            Default
          </Badge>
        ) : null}
      </div>
      {folder.freeSpace > 0 && (
        <p className="text-muted-foreground text-xs">
          <HardDrive className="mr-1 inline size-3" />
          {formatBytes(folder.freeSpace)} free
        </p>
      )}
    </div>
  )
}

function FolderActions({ folder, actions }: { folder: RootFolder; actions: FolderActions }) {
  return (
    <div className="flex items-center gap-2">
      {folder.isDefault ? (
        <Button variant="outline" size="sm" onClick={() => actions.onClearDefault(folder.mediaType)} title="Clear default">
          <X className="mr-1 size-3" />
          Clear Default
        </Button>
      ) : (
        <Button variant="outline" size="sm" onClick={() => actions.onSetDefault(folder.id, folder.mediaType)} title="Set as default">
          <Check className="mr-1 size-3" />
          Set Default
        </Button>
      )}
      <ConfirmDialog
        trigger={
          <Button variant="ghost" size="icon">
            <Trash2 className="size-4" />
          </Button>
        }
        title="Delete root folder"
        description={`Are you sure you want to delete "${folder.name}" (${folder.path})?`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => actions.onDelete(folder.id)}
      />
    </div>
  )
}

function FolderCard({ folder, actions }: { folder: RootFolder; actions: FolderActions }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between py-4">
        <FolderInfo folder={folder} />
        <div className="flex items-center gap-4">
          <FolderBadges folder={folder} />
          <FolderActions folder={folder} actions={actions} />
        </div>
      </CardHeader>
    </Card>
  )
}

type AddFolderState = {
  name: string
  path: string
  mediaType: 'movie' | 'tv'
}

function AddFolderFormBody({
  state,
  onNameChange,
  onPathChange,
  onMediaTypeChange,
  onBrowse,
}: {
  state: AddFolderState
  onNameChange: (v: string) => void
  onPathChange: (v: string) => void
  onMediaTypeChange: (v: 'movie' | 'tv') => void
  onBrowse: () => void
}) {
  return (
    <div className="space-y-4 py-4">
      <div className="space-y-2">
        <Label htmlFor="name">Name</Label>
        <Input id="name" placeholder="Folder name (defaults to directory name)" value={state.name} onChange={(e) => onNameChange(e.target.value)} />
      </div>
      <div className="space-y-2">
        <Label htmlFor="path">Path</Label>
        <div className="flex gap-2">
          <Input id="path" placeholder="/path/to/media or C:\path\to\media" value={state.path} onChange={(e) => onPathChange(e.target.value)} className="flex-1" />
          <Button type="button" variant="outline" size="icon" onClick={onBrowse} title="Browse folders">
            <FolderSearch className="size-4" />
          </Button>
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="mediaType">Media Type</Label>
        <Select value={state.mediaType} onValueChange={(v) => v && onMediaTypeChange(v)}>
          <SelectTrigger>
            <SelectValue>{state.mediaType === 'movie' ? 'Movies' : 'TV Shows'}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="movie">Movies</SelectItem>
            <SelectItem value="tv">TV Shows</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}

function AddFolderDialog({
  open,
  onOpenChange,
  state,
  onNameChange,
  onPathChange,
  onMediaTypeChange,
  onBrowse,
  onAdd,
  isPending,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  state: AddFolderState
  onNameChange: (v: string) => void
  onPathChange: (v: string) => void
  onMediaTypeChange: (v: 'movie' | 'tv') => void
  onBrowse: () => void
  onAdd: () => void
  isPending: boolean
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add Root Folder</DialogTitle>
        </DialogHeader>
        <AddFolderFormBody state={state} onNameChange={onNameChange} onPathChange={onPathChange} onMediaTypeChange={onMediaTypeChange} onBrowse={onBrowse} />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={onAdd} disabled={isPending}>Add</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function useRootFolderActions() {
  const [showAddDialog, setShowAddDialog] = useState(false)
  const [showBrowser, setShowBrowser] = useState(false)
  const [newPath, setNewPath] = useState('')
  const [newName, setNewName] = useState('')
  const [newMediaType, setNewMediaType] = useState<'movie' | 'tv'>('movie')
  const query = useRootFolders()
  const createMutation = useCreateRootFolder()
  const deleteMutation = useDeleteRootFolder()
  const setDefaultMutation = useSetDefault()
  const clearDefaultMutation = useClearDefault()
  const handleAdd = () => {
    if (!newPath.trim()) { toast.error('Please enter a path'); return }
    void withToast(async () => {
      await createMutation.mutateAsync({ path: newPath, name: newName.trim(), mediaType: newMediaType })
      toast.success('Root folder added')
      setShowAddDialog(false)
      setNewPath('')
      setNewName('')
    }, 'Failed to add root folder')()
  }
  const handleDelete = (id: number) => {
    void withToast(async () => {
      await deleteMutation.mutateAsync(id)
      toast.success('Root folder deleted')
    }, 'Failed to delete root folder')()
  }
  const handleSetDefault = (id: number, mediaType: MediaType) => {
    void withToast(async () => {
      await setDefaultMutation.mutateAsync({ entityType: 'root_folder', mediaType, entityId: id })
      toast.success(`Default ${mediaType} root folder set`)
    }, 'Failed to set default root folder')()
  }
  const handleClearDefault = (mediaType: MediaType) => {
    void withToast(async () => {
      await clearDefaultMutation.mutateAsync({ entityType: 'root_folder', mediaType })
      toast.success(`Default ${mediaType} root folder cleared`)
    }, 'Failed to clear default root folder')()
  }
  const folderActions = { onDelete: handleDelete, onSetDefault: handleSetDefault, onClearDefault: handleClearDefault } as FolderActions
  return {
    query, showAddDialog, setShowAddDialog, showBrowser, setShowBrowser,
    newPath, setNewPath, newName, setNewName, newMediaType, setNewMediaType,
    handleAdd, isPending: createMutation.isPending, folderActions,
  }
}

export function RootFoldersSection() {
  const s = useRootFolderActions()
  const { data: folders, isLoading, isError, refetch } = s.query

  return (
    <>
      <ListSection
        data={folders}
        isLoading={isLoading}
        isError={isError}
        refetch={refetch}
        emptyIcon={<FolderOpen className="size-8" />}
        emptyTitle="No root folders"
        emptyDescription="Add a root folder to store your media"
        emptyAction={{ label: 'Add Folder', onClick: () => s.setShowAddDialog(true) }}
        renderItem={(folder) => <FolderCard folder={folder} actions={s.folderActions} />}
        keyExtractor={(folder) => folder.id}
        addPlaceholder={{ label: 'Add Root Folder', onClick: () => s.setShowAddDialog(true) }}
      />
      <AddFolderDialog
        open={s.showAddDialog}
        onOpenChange={s.setShowAddDialog}
        state={{ name: s.newName, path: s.newPath, mediaType: s.newMediaType }}
        onNameChange={s.setNewName}
        onPathChange={s.setNewPath}
        onMediaTypeChange={s.setNewMediaType}
        onBrowse={() => s.setShowBrowser(true)}
        onAdd={s.handleAdd}
        isPending={s.isPending}
      />
      <FolderBrowser open={s.showBrowser} onOpenChange={s.setShowBrowser} initialPath={s.newPath} onSelect={(path) => s.setNewPath(path)} />
    </>
  )
}

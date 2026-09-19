import { FolderSearch } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogBody,
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

import type { useRootFoldersPage } from './use-root-folders-page'

type PageState = ReturnType<typeof useRootFoldersPage>

function AddFolderFormBody({ page }: { page: PageState }) {
  const { state, setName, setPath, setMediaType } = page.newFolder

  return (
    <div className="space-y-4 py-4">
      <div className="space-y-2">
        <Label htmlFor="name">Name</Label>
        <Input
          id="name"
          placeholder="Folder name (defaults to directory name)"
          value={state.name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="path">Path</Label>
        <div className="flex gap-2">
          <Input
            id="path"
            placeholder="/path/to/media or C:\path\to\media"
            value={state.path}
            onChange={(e) => setPath(e.target.value)}
            className="flex-1"
          />
          <Button
            type="button"
            variant="outline"
            size="icon"
            onClick={() => page.setShowBrowser(true)}
            title="Browse folders"
          >
            <FolderSearch className="size-4" />
          </Button>
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="mediaType">Media Type</Label>
        <Select value={state.mediaType} onValueChange={(v) => v && setMediaType(v)}>
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

export function AddRootFolderDialog({ page }: { page: PageState }) {
  return (
    <Dialog open={page.showAddDialog} onOpenChange={page.setShowAddDialog}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add Root Folder</DialogTitle>
        </DialogHeader>
        <DialogBody>
          <AddFolderFormBody page={page} />
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => page.setShowAddDialog(false)}>
            Cancel
          </Button>
          <Button onClick={page.handleAdd} disabled={page.isPending}>
            Add
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

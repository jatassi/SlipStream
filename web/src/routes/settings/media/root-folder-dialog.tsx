import { FolderSearch } from 'lucide-react'

import { FolderBrowser } from '@/components/forms/folder-browser'
import { FormActions, SheetPresenter } from '@/components/presenter'
import { ControlStack, InputRow, SelectRow, StackedRow } from '@/components/settings/control-row'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import type { useRootFoldersPage } from './use-root-folders-page'

type PageState = ReturnType<typeof useRootFoldersPage>

const MEDIA_TYPE_OPTIONS = [
  { value: 'movie', label: 'Movies' },
  { value: 'tv', label: 'TV Shows' },
]

function PathRow({ page }: { page: PageState }) {
  const { state, setPath } = page.newFolder

  return (
    <StackedRow>
      <Label htmlFor="root-folder-path" className="text-body min-w-0 flex-1 font-medium">
        Path
      </Label>
      <div className="flex gap-2">
        <Input
          id="root-folder-path"
          placeholder="/path/to/media or C:\path\to\media"
          value={state.path}
          onChange={(e) => setPath(e.target.value)}
          className="h-11 flex-1 text-base"
        />
        <Button
          type="button"
          variant="outline"
          className="size-11 shrink-0"
          aria-label="Browse folders"
          onClick={() => page.setShowBrowser(true)}
        >
          <FolderSearch className="size-4" />
        </Button>
      </div>
    </StackedRow>
  )
}

export function AddRootFolderDialog({ page }: { page: PageState }) {
  const { state, setName, setMediaType } = page.newFolder

  return (
    <SheetPresenter
      open={page.showAddDialog}
      onOpenChange={page.setShowAddDialog}
      title="Add Root Folder"
      description="Point SlipStream at a folder that holds your library."
      footer={
        <FormActions
          onCancel={() => page.setShowAddDialog(false)}
          confirmLabel="Add"
          onConfirm={page.handleAdd}
          loading={page.isPending}
        />
      }
    >
      <div className="space-y-4 py-2">
        <ControlStack>
          <InputRow
            stacked
            label="Name"
            aria-label="Name"
            placeholder="Folder name (defaults to directory name)"
            value={state.name}
            onChange={(e) => setName(e.target.value)}
          />
          <PathRow page={page} />
          <SelectRow
            label="Media Type"
            value={state.mediaType}
            onChange={(value) => setMediaType(value as 'movie' | 'tv')}
            options={MEDIA_TYPE_OPTIONS}
          />
        </ControlStack>
      </div>
      <FolderBrowser
        nested
        open={page.showBrowser}
        onOpenChange={page.setShowBrowser}
        initialPath={state.path}
        onSelect={(path) => page.newFolder.setPath(path)}
      />
    </SheetPresenter>
  )
}

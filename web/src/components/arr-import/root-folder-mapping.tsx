import { ArrowRight } from 'lucide-react'

import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import type { SourceRootFolder } from '@/types/arr-import'
import type { RootFolder } from '@/types/root-folder'

type RootFolderMappingSectionProps = {
  label: string
  sourceRootFolders: SourceRootFolder[]
  targetRootFolders: RootFolder[]
  rootFolderMapping: Record<string, number>
  setRootFolderMapping: React.Dispatch<React.SetStateAction<Record<string, number>>>
}

export function RootFolderMappingSection({
  label,
  sourceRootFolders,
  targetRootFolders,
  rootFolderMapping,
  setRootFolderMapping,
}: RootFolderMappingSectionProps) {
  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-base font-medium">Root Folder Mapping</h3>
        <p className="text-muted-foreground text-sm">
          Map each {label} root folder to a SlipStream root folder
        </p>
      </div>

      <div className="space-y-3">
        {sourceRootFolders.map((sourceFolder) => (
          <RootFolderMappingRow
            key={sourceFolder.id}
            label={label}
            sourceFolder={sourceFolder}
            targetRootFolders={targetRootFolders}
            selectedTargetId={rootFolderMapping[sourceFolder.path]}
            onSelect={(targetId) =>
              setRootFolderMapping((prev) => ({ ...prev, [sourceFolder.path]: targetId }))
            }
          />
        ))}
      </div>
    </div>
  )
}

function RootFolderMappingRow({
  label,
  sourceFolder,
  targetRootFolders,
  selectedTargetId,
  onSelect,
}: {
  label: string
  sourceFolder: SourceRootFolder
  targetRootFolders: RootFolder[]
  selectedTargetId: number | undefined
  onSelect: (targetId: number) => void
}) {
  const selectedFolder = targetRootFolders.find((f) => f.id === selectedTargetId)
  const triggerDisplay = selectedFolder
    ? `${selectedFolder.name} — ${selectedFolder.path}`
    : 'Select folder...'

  return (
    <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-4">
      <div className="space-y-1">
        <Label className="text-xs font-normal">{label} Root Folder</Label>
        <div className="border-input bg-muted/30 text-muted-foreground rounded-md border px-3 py-2 text-sm">
          {sourceFolder.path}
        </div>
      </div>

      <ArrowRight className="text-muted-foreground mt-5 size-4" />

      <div className="space-y-1">
        <Label className="text-xs font-normal">SlipStream</Label>
        <Select
          value={selectedTargetId?.toString() ?? ''}
          onValueChange={(value) => {
            if (value) {
              onSelect(Number.parseInt(value, 10))
            }
          }}
        >
          <SelectTrigger>{triggerDisplay}</SelectTrigger>
          <SelectContent>
            {targetRootFolders.map((folder) => (
              <SelectItem key={folder.id} value={folder.id.toString()}>
                {folder.name} — {folder.path}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}

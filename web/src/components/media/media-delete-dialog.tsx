import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Checkbox } from '@/components/ui/checkbox'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  selectedCount: number
  deleteFiles: boolean
  onDeleteFilesChange: (checked: boolean) => void
  onConfirm: () => void
  isPending: boolean
  mediaLabel: string
  pluralMediaLabel: string
}

export function MediaDeleteDialog({
  open,
  onOpenChange,
  selectedCount,
  deleteFiles,
  onDeleteFilesChange,
  onConfirm,
  isPending,
  mediaLabel,
  pluralMediaLabel,
}: Props) {
  const label = selectedCount > 1 ? pluralMediaLabel : mediaLabel

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            Delete {selectedCount} {label}?
          </AlertDialogTitle>
          <AlertDialogDescription>
            This action cannot be undone. The selected {pluralMediaLabel.toLowerCase()} will be removed from your library.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="flex items-center gap-2 py-2">
          <Checkbox
            id={`delete${mediaLabel}Files`}
            checked={deleteFiles}
            onCheckedChange={(checked) => onDeleteFilesChange(checked)}
          />
          <label htmlFor={`delete${mediaLabel}Files`} className="cursor-pointer text-sm">
            Also delete files from disk
          </label>
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={onConfirm} disabled={isPending}>
            {isPending ? 'Deleting...' : 'Delete'}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

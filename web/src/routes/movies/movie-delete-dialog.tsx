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

type MovieDeleteDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  selectedCount: number
  deleteFiles: boolean
  onDeleteFilesChange: (checked: boolean) => void
  onConfirm: () => void
  isPending: boolean
}

export function MovieDeleteDialog({
  open,
  onOpenChange,
  selectedCount,
  deleteFiles,
  onDeleteFilesChange,
  onConfirm,
  isPending,
}: MovieDeleteDialogProps) {
  const plural = selectedCount > 1 ? 's' : ''

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            Delete {selectedCount} Movie{plural}?
          </AlertDialogTitle>
          <AlertDialogDescription>
            This action cannot be undone. The selected movies will be removed from your library.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="flex items-center gap-2 py-2">
          <Checkbox
            id="deleteFiles"
            checked={deleteFiles}
            onCheckedChange={(checked) => onDeleteFilesChange(checked)}
          />
          <label htmlFor="deleteFiles" className="cursor-pointer text-sm">
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

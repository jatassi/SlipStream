import type { SearchIndexerError } from '@/types'

export function SearchFooter({
  total,
  indexersSearched,
  errors,
}: {
  total: number
  indexersSearched: number
  errors: SearchIndexerError[]
}) {
  return (
    <div className="text-muted-foreground flex items-center justify-between border-t pt-4 text-sm">
      <span>
        {total} release{total === 1 ? '' : 's'} from {indexersSearched} indexer
        {indexersSearched === 1 ? '' : 's'}
      </span>
      {errors.length > 0 && (
        <span className="text-destructive">
          {errors.length} error{errors.length === 1 ? '' : 's'}
        </span>
      )}
    </div>
  )
}

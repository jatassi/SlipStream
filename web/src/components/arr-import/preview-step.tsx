import { CheckCircle2, Film, Loader2, SkipForward, Tv, XCircle } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { ImportPreview, MoviePreview, SeriesPreview } from '@/types/arr-import'

type PreviewStepProps = {
  preview: ImportPreview
  onStartImport: () => void
  isImporting: boolean
}

function SummaryCard({ label, value, icon: Icon }: { label: string; value: number; icon: React.ElementType }) {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-border bg-muted/50 p-4">
      <Icon className="size-5 text-muted-foreground" />
      <div>
        <div className="text-2xl font-semibold">{value}</div>
        <div className="text-sm text-muted-foreground">{label}</div>
      </div>
    </div>
  )
}

function MoviePreviewItem({ movie }: { movie: MoviePreview }) {
  const statusBadge = {
    new: { variant: 'default' as const, label: 'New' },
    duplicate: { variant: 'secondary' as const, label: 'Duplicate' },
    skip: { variant: 'destructive' as const, label: 'Skip' },
  }[movie.status]

  return (
    <div className="flex items-center justify-between border-b border-border py-3 last:border-0">
      <div className="flex-1">
        <div className="font-medium">
          {movie.title} ({movie.year})
        </div>
        {movie.hasFile ? <div className="text-sm text-muted-foreground">Quality: {movie.quality}</div> : null}
        {movie.skipReason ? <div className="text-sm text-destructive">{movie.skipReason}</div> : null}
      </div>
      <Badge variant={statusBadge.variant}>{statusBadge.label}</Badge>
    </div>
  )
}

function SeriesPreviewItem({ series }: { series: SeriesPreview }) {
  const statusBadge = {
    new: { variant: 'default' as const, label: 'New' },
    duplicate: { variant: 'secondary' as const, label: 'Duplicate' },
    skip: { variant: 'destructive' as const, label: 'Skip' },
  }[series.status]

  return (
    <div className="flex items-center justify-between border-b border-border py-3 last:border-0">
      <div className="flex-1">
        <div className="font-medium">
          {series.title} ({series.year})
        </div>
        <div className="text-sm text-muted-foreground">
          {series.episodeCount} episode{series.episodeCount === 1 ? '' : 's'} • {series.fileCount} file
          {series.fileCount === 1 ? '' : 's'}
        </div>
        {series.skipReason ? <div className="text-sm text-destructive">{series.skipReason}</div> : null}
      </div>
      <Badge variant={statusBadge.variant}>{statusBadge.label}</Badge>
    </div>
  )
}

function usePreviewSummary(preview: ImportPreview) {
  const { summary, movies } = preview
  const isMovieImport = movies.length > 0

  return {
    isMovieImport,
    totalItems: isMovieImport ? summary.totalMovies : summary.totalSeries,
    newItems: isMovieImport ? summary.newMovies : summary.newSeries,
    duplicateItems: isMovieImport ? summary.duplicateMovies : summary.duplicateSeries,
    skippedItems: isMovieImport ? summary.skippedMovies : summary.skippedSeries,
  }
}

function PreviewSummaryCards({ preview }: { preview: ImportPreview }) {
  const { isMovieImport, totalItems, newItems, duplicateItems, skippedItems } = usePreviewSummary(preview)

  return (
    <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
      <SummaryCard label={isMovieImport ? 'Movies' : 'Series'} value={totalItems} icon={isMovieImport ? Film : Tv} />
      <SummaryCard label="New" value={newItems} icon={CheckCircle2} />
      <SummaryCard label="Duplicates" value={duplicateItems} icon={XCircle} />
      <SummaryCard label="Skipped" value={skippedItems} icon={SkipForward} />
    </div>
  )
}

function PreviewItemList({ preview }: { preview: ImportPreview }) {
  const { isMovieImport, totalItems } = usePreviewSummary(preview)

  return (
    <div className="space-y-2">
      <h3 className="text-lg font-semibold">
        {isMovieImport ? 'Movies' : 'Series'} ({totalItems})
      </h3>
      <ScrollArea className="h-[50vh] rounded-lg border border-border bg-background p-4">
        {isMovieImport
          ? preview.movies.map((movie) => <MoviePreviewItem key={movie.tmdbId} movie={movie} />)
          : preview.series.map((s) => <SeriesPreviewItem key={s.tvdbId} series={s} />)}
      </ScrollArea>
    </div>
  )
}

function ImportButton({ onStartImport, isImporting, preview }: PreviewStepProps) {
  const { isMovieImport, newItems } = usePreviewSummary(preview)
  const label = `Import ${newItems} ${isMovieImport ? 'Movie' : 'Series'}${newItems === 1 ? '' : 's'}`

  return (
    <div className="flex justify-end">
      <Button onClick={onStartImport} disabled={isImporting || newItems === 0} size="lg">
        {isImporting ? (
          <>
            <Loader2 className="size-4 animate-spin" />
            Importing...
          </>
        ) : (
          label
        )}
      </Button>
    </div>
  )
}

export function PreviewStep({ preview, onStartImport, isImporting }: PreviewStepProps) {
  return (
    <div className="space-y-6">
      <PreviewSummaryCards preview={preview} />
      <PreviewItemList preview={preview} />
      <ImportButton preview={preview} onStartImport={onStartImport} isImporting={isImporting} />
    </div>
  )
}

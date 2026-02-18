import { useState } from 'react'

import { Link } from '@tanstack/react-router'
import { AlertCircle, CheckCircle2, Film, Loader2, Tv } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Progress, ProgressLabel, ProgressValue } from '@/components/ui/progress'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useImportProgress } from '@/hooks/use-arr-import'
import { cn } from '@/lib/utils'
import type { ImportReport, SourceType } from '@/types/arr-import'
import type { Activity } from '@/types/progress'

type ImportStepProps = {
  onDone: () => void
  sourceType: SourceType
}

function SummaryCard({
  label,
  value,
  icon: Icon,
  alert,
}: {
  label: string
  value: number
  icon: React.ElementType
  alert?: boolean
}) {
  return (
    <div
      className={cn(
        'flex items-center gap-3 rounded-lg border p-4',
        alert ? 'border-destructive bg-destructive/10' : 'border-border bg-muted/50',
      )}
    >
      <Icon className={cn('size-5', alert ? 'text-destructive' : 'text-muted-foreground')} />
      <div>
        <div className="text-2xl font-semibold">{value}</div>
        <div className="text-sm text-muted-foreground">{label}</div>
      </div>
    </div>
  )
}

function InProgressView({
  title,
  subtitle,
  progressValue,
  sourceType,
}: {
  title: string
  subtitle?: string
  progressValue: number
  sourceType: SourceType
}) {
  const isIndeterminate = progressValue === -1

  return (
    <div className="space-y-6">
      <div className="flex flex-col items-center justify-center gap-6 py-12">
        <Loader2 className="size-12 animate-spin text-muted-foreground" />
        <div className="space-y-2 text-center">
          <h3 className="text-lg font-semibold">{title}</h3>
          {subtitle ? <p className="text-sm text-muted-foreground">{subtitle}</p> : null}
        </div>
        <div className="w-full max-w-md">
          <Progress
            value={isIndeterminate ? null : progressValue}
            variant={sourceType === 'radarr' ? 'movie' : 'tv'}
            className="w-full"
          >
            <ProgressLabel>Import Progress</ProgressLabel>
            <ProgressValue />
          </Progress>
        </div>
      </div>
    </div>
  )
}

function ViewLibraryLink({ sourceType }: { sourceType: SourceType }) {
  const isMovie = sourceType === 'radarr'
  return (
    <Button
      render={<Link to={isMovie ? '/movies' : '/series'} />}
      size="lg"
      className={isMovie ? 'bg-movie-500 hover:bg-movie-600' : 'bg-tv-500 hover:bg-tv-600'}
    >
      {isMovie ? 'View Movies' : 'View Series'}
    </Button>
  )
}

function CompletionView({ onDone, sourceType }: { onDone: () => void; sourceType: SourceType }) {
  return (
    <div className="flex flex-col items-center justify-center gap-6 py-12">
      <CheckCircle2 className="size-12 text-green-500" />
      <div className="space-y-2 text-center">
        <h3 className="text-lg font-semibold">Import Complete</h3>
        <p className="text-sm text-muted-foreground">The import has finished successfully.</p>
      </div>
      <div className="flex items-center gap-3">
        <ViewLibraryLink sourceType={sourceType} />
        <Button onClick={onDone} size="lg" variant="outline">
          Done
        </Button>
      </div>
    </div>
  )
}

function ErrorList({ errors }: { errors: string[] }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <h4 className="font-medium">Errors</h4>
        <Badge variant="destructive">{errors.length}</Badge>
      </div>
      <ScrollArea className="h-48 rounded-lg border border-border bg-background p-4">
        <div className="space-y-2">
          {errors.map((error) => (
            <div key={error} className="flex gap-2 border-b border-border pb-2 last:border-0">
              <AlertCircle className="mt-0.5 size-4 shrink-0 text-destructive" />
              <p className="text-sm text-muted-foreground">{error}</p>
            </div>
          ))}
        </div>
      </ScrollArea>
    </div>
  )
}

function MovieSummary({ report }: { report: ImportReport }) {
  return (
    <>
      <SummaryCard label="Movies Created" value={report.moviesCreated} icon={Film} />
      <SummaryCard label="Movies Skipped" value={report.moviesSkipped} icon={Film} />
      {report.moviesErrored > 0 ? (
        <SummaryCard label="Movies Errored" value={report.moviesErrored} icon={AlertCircle} />
      ) : null}
    </>
  )
}

function SeriesSummary({ report }: { report: ImportReport }) {
  return (
    <>
      <SummaryCard label="Series Created" value={report.seriesCreated} icon={Tv} />
      <SummaryCard label="Series Skipped" value={report.seriesSkipped} icon={Tv} />
      {report.seriesErrored > 0 ? (
        <SummaryCard label="Series Errored" value={report.seriesErrored} icon={AlertCircle} />
      ) : null}
    </>
  )
}

function ReportView({
  report,
  onDone,
  sourceType,
}: {
  report: ImportReport
  onDone: () => void
  sourceType: SourceType
}) {
  const hasErrors = report.errors.length > 0
  const isMovieImport = sourceType === 'radarr'

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          {hasErrors ? (
            <AlertCircle className="size-6 text-yellow-500" />
          ) : (
            <CheckCircle2 className="size-6 text-green-500" />
          )}
          <h3 className="text-lg font-semibold">
            Import {hasErrors ? 'Completed with Errors' : 'Complete'}
          </h3>
        </div>

        <div className="grid grid-cols-2 gap-4 md:grid-cols-3">
          {isMovieImport ? <MovieSummary report={report} /> : <SeriesSummary report={report} />}
          <SummaryCard
            label={`Files Imported (${report.filesImported}/${report.totalFiles})`}
            value={report.filesImported}
            icon={report.filesImported < report.totalFiles ? AlertCircle : CheckCircle2}
            alert={report.filesImported < report.totalFiles}
          />
        </div>
      </div>

      {hasErrors ? <ErrorList errors={report.errors} /> : null}

      <div className="flex justify-end gap-3">
        <ViewLibraryLink sourceType={sourceType} />
        <Button onClick={onDone} size="lg" variant="outline">
          Done
        </Button>
      </div>
    </div>
  )
}

const TERMINAL_STATUSES = new Set(['completed', 'failed', 'cancelled'])

function useImportResult() {
  const progress = useImportProgress()
  const [finished, setFinished] = useState<Activity | null>(null)
  const [prevProgress, setPrevProgress] = useState(progress)

  if (progress !== prevProgress) {
    setPrevProgress(progress)
    if (progress && TERMINAL_STATUSES.has(progress.status)) {
      setFinished(progress)
    }
  }

  return { progress, finished }
}

export function ImportStep({ onDone, sourceType }: ImportStepProps) {
  const { progress, finished } = useImportResult()

  if (!finished) {
    return (
      <InProgressView
        title={progress?.title ?? 'Importing...'}
        subtitle={progress?.subtitle}
        progressValue={progress?.progress ?? -1}
        sourceType={sourceType}
      />
    )
  }

  const report = finished.metadata.report as ImportReport | undefined

  if (!report) {
    return <CompletionView onDone={onDone} sourceType={sourceType} />
  }

  return <ReportView report={report} onDone={onDone} sourceType={sourceType} />
}

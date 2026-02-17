import { AlertCircle, CheckCircle2, Film, Loader2, Tv } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Progress, ProgressLabel, ProgressValue } from '@/components/ui/progress'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useImportProgress } from '@/hooks/use-arr-import'
import type { ImportReport } from '@/types/arr-import'

type ImportStepProps = {
  onDone: () => void
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

function InProgressView({ title, subtitle, progressValue }: { title: string; subtitle?: string; progressValue: number }) {
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
          <Progress value={isIndeterminate ? null : progressValue} variant="media" className="w-full">
            <ProgressLabel>Import Progress</ProgressLabel>
            <ProgressValue />
          </Progress>
        </div>
      </div>
    </div>
  )
}

function CompletionView({ onDone }: { onDone: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center gap-6 py-12">
      <CheckCircle2 className="size-12 text-green-500" />
      <div className="space-y-2 text-center">
        <h3 className="text-lg font-semibold">Import Complete</h3>
        <p className="text-sm text-muted-foreground">The import has finished successfully.</p>
      </div>
      <Button onClick={onDone} size="lg">
        Done
      </Button>
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

function ReportView({ report, onDone }: { report: ImportReport; onDone: () => void }) {
  const hasErrors = report.errors.length > 0
  const totalMovies = report.moviesCreated + report.moviesSkipped + report.moviesErrored
  const isMovieImport = totalMovies > 0

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
          <SummaryCard label="Files Imported" value={report.filesImported} icon={CheckCircle2} />
        </div>
      </div>

      {hasErrors ? <ErrorList errors={report.errors} /> : null}

      <div className="flex justify-end">
        <Button onClick={onDone} size="lg">
          Done
        </Button>
      </div>
    </div>
  )
}

export function ImportStep({ onDone }: ImportStepProps) {
  const progress = useImportProgress()

  if (!progress || progress.status === 'pending' || progress.status === 'in_progress') {
    return (
      <InProgressView
        title={progress?.title ?? 'Importing...'}
        subtitle={progress?.subtitle}
        progressValue={progress?.progress ?? -1}
      />
    )
  }

  const report = progress.metadata.report as ImportReport | undefined

  if (!report) {
    return <CompletionView onDone={onDone} />
  }

  return <ReportView report={report} onDone={onDone} />
}

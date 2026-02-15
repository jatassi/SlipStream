import { useMemo, useState } from 'react'

import { Bug, ChevronDown, ChevronUp } from 'lucide-react'
import { marked } from 'marked'

import { PageHeader } from '@/components/layout/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import type { UpdateState } from '@/types/update'

import { UpdateStateDisplay } from './update-state-display'
import { useUpdatePage } from './use-update-page'

function DebugButton({ state, onClick }: { state: UpdateState; onClick: () => void }) {
  return (
    <Button variant="outline" size="sm" onClick={onClick} title={`Current: ${state}`}>
      <Bug className="mr-2 size-4" />
      Debug: {state}
    </Button>
  )
}

function ReleaseNotes({ notes }: { notes: string }) {
  const [expanded, setExpanded] = useState(false)
  const lines = notes.split('\n')
  const previewLines = 8
  const hasMore = lines.length > previewLines
  const displayedContent = expanded ? notes : lines.slice(0, previewLines).join('\n')

  const renderedMarkdown = useMemo(() => {
    return marked.parse(displayedContent, { async: false })
  }, [displayedContent])

  return (
    <div className="space-y-3">
      <div className="text-muted-foreground text-sm font-medium">Release Notes</div>
      <MarkdownContent html={renderedMarkdown} expanded={expanded} hasMore={hasMore} />
      {hasMore ? (
        <Button variant="ghost" size="sm" onClick={() => setExpanded(!expanded)} className="w-full">
          {expanded ? <ChevronUp className="mr-1 size-4" /> : <ChevronDown className="mr-1 size-4" />}
          {expanded ? 'Show Less' : 'Show More'}
        </Button>
      ) : null}
    </div>
  )
}

function MarkdownContent({
  html,
  expanded,
  hasMore,
}: {
  html: string
  expanded: boolean
  hasMore: boolean
}) {
  return (
    <div
      className={cn(
        'bg-muted/50 relative rounded-lg p-4 text-sm',
        !expanded && hasMore && 'max-h-48 overflow-hidden',
      )}
    >
      <div
        className="prose prose-sm prose-invert [&_h2]:text-foreground [&_h3]:text-foreground [&_li]:text-foreground/80 [&_strong]:text-foreground [&_p]:text-foreground/80 [&_code]:bg-muted max-w-none [&_code]:rounded [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs [&_h2]:mt-4 [&_h2]:mb-2 [&_h2]:text-base [&_h2]:font-semibold [&_h2:first-child]:mt-0 [&_h3]:mt-3 [&_h3]:mb-1.5 [&_h3]:text-sm [&_h3]:font-semibold [&_li]:my-0.5 [&_p]:my-1.5 [&_strong]:font-semibold [&_ul]:my-1.5 [&_ul]:pl-4"
        dangerouslySetInnerHTML={{ __html: html }}
      />
      {!expanded && hasMore ? (
        <div className="from-muted/50 absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t to-transparent" />
      ) : null}
    </div>
  )
}

export function UpdatePage() {
  const page = useUpdatePage()

  return (
    <div>
      <PageHeader
        title="Software Update"
        description="Check for and install SlipStream updates"
        actions={
          page.developerMode ? (
            <DebugButton state={page.state} onClick={page.cycleDebugState} />
          ) : null
        }
      />
      <div className="max-w-lg">
        <Card>
          <CardContent className="py-1">
            <UpdateStateDisplay
              state={page.state}
              currentVersion={page.currentVersion}
              newVersion={page.newVersion}
              progress={page.progress}
              error={page.error}
              onCheckForUpdate={page.handleCheckForUpdate}
              onDownloadUpdate={page.handleDownloadUpdate}
              onRetry={page.handleRetry}
              downloadedMB={page.downloadedMB}
              totalMB={page.totalMB}
              isChecking={page.isChecking}
              isInstalling={page.isInstalling}
            />
          </CardContent>
        </Card>
        {page.showReleaseNotes && page.releaseNotes ? (
          <Card className="mt-4">
            <CardContent className="py-1">
              <ReleaseNotes notes={page.releaseNotes} />
            </CardContent>
          </Card>
        ) : null}
      </div>
    </div>
  )
}

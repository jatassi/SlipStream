import { AlertTriangle, CheckCircle2, ExternalLink } from 'lucide-react'

import { useMediainfoAvailable } from '@/hooks'

export function MediaInfoStatus() {
  const isAvailable = useMediainfoAvailable()

  if (isAvailable) {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-green-500/20 bg-green-500/10 p-3">
        <CheckCircle2 className="size-4 shrink-0 text-green-500" />
        <span className="text-sm">MediaInfo is installed and available for file probing</span>
      </div>
    )
  }

  return (
    <div className="flex items-center justify-between gap-4 rounded-lg border border-amber-500/20 bg-amber-500/10 p-3">
      <div className="flex items-center gap-2">
        <AlertTriangle className="size-4 shrink-0 text-amber-500" />
        <span className="text-sm">
          MediaInfo not found - file probing will use filename parsing only
        </span>
      </div>
      <a
        href="https://mediaarea.net/en/MediaInfo/Download"
        target="_blank"
        rel="noopener noreferrer"
        className="bg-primary text-primary-foreground hover:bg-primary/90 inline-flex shrink-0 items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
      >
        Download
        <ExternalLink className="size-3" />
      </a>
    </div>
  )
}

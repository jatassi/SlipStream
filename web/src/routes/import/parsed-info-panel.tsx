import { Badge } from '@/components/ui/badge'
import type { ParsedMediaInfo } from '@/types'

function InfoRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex justify-between">
      <span className="text-muted-foreground">{label}:</span>
      {children}
    </div>
  )
}

function ParsedDetails({ parsed }: { parsed: ParsedMediaInfo }) {
  const endEpSuffix =
    parsed.endEpisode && parsed.endEpisode !== parsed.episode ? `-${parsed.endEpisode}` : ''

  return (
    <div className="space-y-3">
      <h4 className="text-muted-foreground text-sm font-medium">Parsed Information</h4>
      <div className="space-y-2 text-sm">
        {parsed.title ? <InfoRow label="Title"><span className="font-medium">{parsed.title}</span></InfoRow> : null}
        {parsed.year ? <InfoRow label="Year"><span>{parsed.year}</span></InfoRow> : null}
        {parsed.isTV ? (
          <>
            <InfoRow label="Season"><span>{parsed.season}</span></InfoRow>
            <InfoRow label="Episode"><span>{parsed.episode}{endEpSuffix}</span></InfoRow>
          </>
        ) : null}
      </div>
    </div>
  )
}

function QualityDetails({ parsed }: { parsed: ParsedMediaInfo }) {
  const hasAudio = parsed.audioCodecs && parsed.audioCodecs.length > 0

  return (
    <div className="space-y-3">
      <h4 className="text-muted-foreground text-sm font-medium">Quality Information</h4>
      <div className="space-y-2 text-sm">
        {parsed.quality ? <InfoRow label="Quality"><Badge variant="secondary">{parsed.quality}</Badge></InfoRow> : null}
        {parsed.source ? <InfoRow label="Source"><span>{parsed.source}</span></InfoRow> : null}
        {parsed.codec ? <InfoRow label="Codec"><span>{parsed.codec}</span></InfoRow> : null}
        {hasAudio ? <InfoRow label="Audio"><span>{parsed.audioCodecs?.join(', ')}</span></InfoRow> : null}
      </div>
    </div>
  )
}

export function ParsedInfoPanel({ parsed }: { parsed: ParsedMediaInfo }) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <ParsedDetails parsed={parsed} />
      <QualityDetails parsed={parsed} />
    </div>
  )
}

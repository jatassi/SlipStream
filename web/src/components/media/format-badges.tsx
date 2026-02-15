import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

type FormatBadgesProps = {
  source?: string
  codec?: string
  attributes: string[]
  className?: string
}

// Map codec values to display names
const codecDisplayNames: Record<string, string> = {
  x265: 'HEVC',
  x264: 'AVC',
  AV1: 'AV1',
  VP9: 'VP9',
  XviD: 'XviD',
  DivX: 'DivX',
  MPEG2: 'MPEG2',
}

// Attribute styling based on type
function getAttributeStyle(attr: string): string {
  const attrLower = attr.toLowerCase()

  // HDR attributes - purple/violet tones
  if (['dv', 'hdr10+', 'hdr10', 'hdr', 'hlg'].includes(attrLower)) {
    return 'bg-violet-500/20 text-violet-400 border-violet-500/30'
  }

  // Audio attributes - blue tones
  if (
    ['atmos', 'dts-x', 'dts-hd', 'truehd', 'dts', 'dd+', 'dd', 'aac', 'flac'].includes(attrLower)
  ) {
    return 'bg-blue-500/20 text-blue-400 border-blue-500/30'
  }

  // REMUX - gold/amber tone
  if (attrLower === 'remux') {
    return 'bg-amber-500/20 text-amber-400 border-amber-500/30'
  }

  // Default styling
  return ''
}

export function FormatBadges({ source, codec, attributes, className }: FormatBadgesProps) {
  const badges: { label: string; style?: string }[] = []

  // Add codec badge (use display name)
  if (codec) {
    const displayName = codecDisplayNames[codec] || codec
    badges.push({ label: displayName })
  }

  // Add attribute badges
  for (const attr of attributes) {
    badges.push({
      label: attr,
      style: getAttributeStyle(attr),
    })
  }

  // Add source badge if not in attributes (e.g., BluRay, WEB-DL)
  if (source && !attributes.includes('REMUX')) {
    // Only show source if it's not already represented
    badges.push({ label: source })
  }

  if (badges.length === 0) {
    return null
  }

  return (
    <div className={cn('flex flex-wrap gap-1', className)}>
      {badges.map((badge) => (
        <Badge key={badge.label} variant="secondary" className={cn('font-mono text-xs', badge.style)}>
          {badge.label}
        </Badge>
      ))}
    </div>
  )
}

import { Check, CheckCircle, Clock, Download, Library } from 'lucide-react'

import { Badge } from '@/components/ui/badge'

const BADGE_CLASS = 'px-1.5 py-0.5 text-[10px] text-white md:px-2 md:text-xs'
const ICON_CLASS = 'mr-0.5 size-2.5 md:mr-1 md:size-3'

type StatusBadgeProps = {
  hasActiveDownload: boolean
  isInLibrary: boolean
  hasExistingRequest: boolean
  isAvailable: boolean
  isApproved: boolean
}

export function StatusBadge({
  hasActiveDownload,
  isInLibrary,
  hasExistingRequest,
  isAvailable,
  isApproved,
}: StatusBadgeProps) {
  if (hasActiveDownload) {
    return (
      <Badge variant="secondary" className={`bg-purple-600 ${BADGE_CLASS}`}>
        <Download className={ICON_CLASS} />
        Downloading
      </Badge>
    )
  }

  if (isInLibrary) {
    return (
      <Badge variant="secondary" className={`bg-green-600 ${BADGE_CLASS}`}>
        <Library className={ICON_CLASS} />
        In Library
      </Badge>
    )
  }

  if (!hasExistingRequest) {
    return null
  }

  if (isAvailable) {
    return (
      <Badge variant="secondary" className={`bg-green-600 ${BADGE_CLASS}`}>
        <CheckCircle className={ICON_CLASS} />
        Available
      </Badge>
    )
  }

  if (isApproved) {
    return (
      <Badge variant="secondary" className={`bg-blue-600 ${BADGE_CLASS}`}>
        <Check className={ICON_CLASS} />
        Approved
      </Badge>
    )
  }

  return (
    <Badge variant="secondary" className={`bg-yellow-600 ${BADGE_CLASS}`}>
      <Clock className={ICON_CLASS} />
      Requested
    </Badge>
  )
}

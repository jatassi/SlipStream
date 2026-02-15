import type { ReactNode } from 'react'
import { useState } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { Check, CheckCircle, Clock, Library, Plus } from 'lucide-react'

import { NetworkLogo } from '@/components/media/network-logo'
import { PosterImage } from '@/components/media/poster-image'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { RequestStatus, SeriesSearchResult } from '@/types'

import { MediaInfoModal } from './media-info-modal'

type RequestInfo = {
  id: number
  status: RequestStatus
}

const BADGE_CLASS = 'px-1.5 py-0.5 text-[10px] text-white md:px-2 md:text-xs'
const ICON_CLASS = 'mr-0.5 size-2.5 md:mr-1 md:size-3'

const STATUS_BADGE_MAP: Partial<Record<string, { bg: string, icon: typeof Check, label: string }>> = {
  available: { bg: 'bg-green-600', icon: CheckCircle, label: 'Available' },
  approved: { bg: 'bg-blue-600', icon: Check, label: 'Approved' },
  pending: { bg: 'bg-yellow-600', icon: Clock, label: 'Requested' },
}

function getStatusBadge(inLibrary?: boolean, requestInfo?: RequestInfo): ReactNode {
  if (inLibrary) {
    return (
      <Badge variant="secondary" className={`bg-green-600 ${BADGE_CLASS}`}>
        <Library className={ICON_CLASS} />
        In Library
      </Badge>
    )
  }
  if (!requestInfo) {
    return null
  }
  const config = STATUS_BADGE_MAP[requestInfo.status]
  if (!config) {
    return null
  }
  return (
    <Badge variant="secondary" className={`${config.bg} ${BADGE_CLASS}`}>
      <config.icon className={ICON_CLASS} />
      {config.label}
    </Badge>
  )
}

function CardPoster({ series, statusBadge, onClick }: {
  series: SeriesSearchResult
  statusBadge: ReactNode
  onClick: () => void
}) {
  return (
    <button type="button" className="relative aspect-[2/3] cursor-pointer w-full" onClick={onClick}>
      <PosterImage url={series.posterUrl} alt={series.title} type="series" className="absolute inset-0" />
      <div className="absolute top-2 left-2 flex flex-col gap-1">{statusBadge}</div>
      <NetworkLogo logoUrl={series.networkLogoUrl} network={series.network} className="absolute top-2 right-2" />
      <div className="absolute inset-0 bg-black/40 opacity-0 transition-opacity group-hover:opacity-100" />
      <div className="absolute inset-x-0 bottom-0 flex flex-col justify-end p-3 opacity-0 transition-opacity group-hover:opacity-100">
        <h3 className="line-clamp-3 font-semibold text-white">{series.title}</h3>
        <div className="flex items-center gap-2 text-sm text-gray-300">
          <span>{series.year ?? 'Unknown year'}</span>
          {series.network && !series.networkLogoUrl ? (
            <Badge variant="secondary" className="text-xs">{series.network}</Badge>
          ) : null}
        </div>
      </div>
    </button>
  )
}

function CardAction({ inLibrary, onAdd }: { inLibrary?: boolean, onAdd: () => void }) {
  if (inLibrary) {
    return (
      <Button variant="secondary" size="sm" className="w-full text-xs md:text-sm" disabled>
        <Check className="mr-1 size-3 md:mr-2 md:size-4" />
        In Library
      </Button>
    )
  }
  return (
    <Button variant="default" size="sm" className="w-full text-xs md:text-sm" onClick={onAdd}>
      <Plus className="mr-1 size-3 md:mr-2 md:size-4" />
      Add...
    </Button>
  )
}

type ExternalSeriesCardProps = {
  series: SeriesSearchResult
  inLibrary?: boolean
  requestInfo?: RequestInfo
  className?: string
}

export function ExternalSeriesCard({ series, inLibrary, requestInfo, className }: ExternalSeriesCardProps) {
  const navigate = useNavigate()
  const [infoOpen, setInfoOpen] = useState(false)
  const handleAdd = () => void navigate({ to: '/series/add', search: { tmdbId: series.tmdbId } })
  const statusBadge = getStatusBadge(inLibrary, requestInfo)

  return (
    <div className={cn('group bg-card border-border hover:border-tv-500/50 hover:glow-tv overflow-hidden rounded-lg border transition-all', className)}>
      <CardPoster series={series} statusBadge={statusBadge} onClick={() => setInfoOpen(true)} />
      <div className="p-2">
        <CardAction inLibrary={inLibrary} onAdd={handleAdd} />
      </div>
      <MediaInfoModal
        open={infoOpen}
        onOpenChange={setInfoOpen}
        media={series}
        mediaType="series"
        inLibrary={inLibrary}
        onAction={inLibrary ? undefined : handleAdd}
        actionLabel="Add..."
        actionIcon={<Plus className="mr-1 size-3 md:mr-2 md:size-4" />}
        disabledLabel="Already Added"
      />
    </div>
  )
}

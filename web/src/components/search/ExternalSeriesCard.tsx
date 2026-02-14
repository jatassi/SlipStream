import { useState } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { Check, CheckCircle, Clock, Library, Plus } from 'lucide-react'

import { NetworkLogo } from '@/components/media/NetworkLogo'
import { PosterImage } from '@/components/media/PosterImage'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { RequestStatus, SeriesSearchResult } from '@/types'

import { MediaInfoModal } from './MediaInfoModal'

type RequestInfo = {
  id: number
  status: RequestStatus
}

type ExternalSeriesCardProps = {
  series: SeriesSearchResult
  inLibrary?: boolean
  requestInfo?: RequestInfo
  className?: string
}

export function ExternalSeriesCard({
  series,
  inLibrary,
  requestInfo,
  className,
}: ExternalSeriesCardProps) {
  const navigate = useNavigate()
  const [infoOpen, setInfoOpen] = useState(false)

  const handleAdd = () => {
    navigate({ to: '/series/add', search: { tmdbId: series.tmdbId } })
  }

  const isApproved = requestInfo?.status === 'approved'
  const isPending = requestInfo?.status === 'pending'
  const isAvailable = requestInfo?.status === 'available'

  return (
    <div
      className={cn(
        'group bg-card border-border hover:border-tv-500/50 hover:glow-tv overflow-hidden rounded-lg border transition-all',
        className,
      )}
    >
      <div className="relative aspect-[2/3] cursor-pointer" onClick={() => setInfoOpen(true)}>
        <PosterImage
          url={series.posterUrl}
          alt={series.title}
          type="series"
          className="absolute inset-0"
        />

        {/* Status badges */}
        <div className="absolute top-2 left-2 flex flex-col gap-1">
          {inLibrary ? (
            <Badge
              variant="secondary"
              className="bg-green-600 px-1.5 py-0.5 text-[10px] text-white md:px-2 md:text-xs"
            >
              <Library className="mr-0.5 size-2.5 md:mr-1 md:size-3" />
              In Library
            </Badge>
          ) : (
            requestInfo &&
            (isAvailable ? (
              <Badge
                variant="secondary"
                className="bg-green-600 px-1.5 py-0.5 text-[10px] text-white md:px-2 md:text-xs"
              >
                <CheckCircle className="mr-0.5 size-2.5 md:mr-1 md:size-3" />
                Available
              </Badge>
            ) : isApproved ? (
              <Badge
                variant="secondary"
                className="bg-blue-600 px-1.5 py-0.5 text-[10px] text-white md:px-2 md:text-xs"
              >
                <Check className="mr-0.5 size-2.5 md:mr-1 md:size-3" />
                Approved
              </Badge>
            ) : (
              isPending && (
                <Badge
                  variant="secondary"
                  className="bg-yellow-600 px-1.5 py-0.5 text-[10px] text-white md:px-2 md:text-xs"
                >
                  <Clock className="mr-0.5 size-2.5 md:mr-1 md:size-3" />
                  Requested
                </Badge>
              )
            ))
          )}
        </div>

        {/* Network logo */}
        <NetworkLogo
          logoUrl={series.networkLogoUrl}
          network={series.network}
          className="absolute top-2 right-2"
        />

        {/* Hover overlay */}
        <div className="absolute inset-0 bg-black/40 opacity-0 transition-opacity group-hover:opacity-100" />
        <div className="absolute inset-x-0 bottom-0 flex flex-col justify-end p-3 opacity-0 transition-opacity group-hover:opacity-100">
          <h3 className="line-clamp-3 font-semibold text-white">{series.title}</h3>
          <div className="flex items-center gap-2 text-sm text-gray-300">
            <span>{series.year || 'Unknown year'}</span>
            {series.network && !series.networkLogoUrl ? (
              <Badge variant="secondary" className="text-xs">
                {series.network}
              </Badge>
            ) : null}
          </div>
        </div>
      </div>

      <div className="p-2">
        {inLibrary ? (
          <Button variant="secondary" size="sm" className="w-full text-xs md:text-sm" disabled>
            <Check className="mr-1 size-3 md:mr-2 md:size-4" />
            In Library
          </Button>
        ) : (
          <Button
            variant="default"
            size="sm"
            className="w-full text-xs md:text-sm"
            onClick={handleAdd}
          >
            <Plus className="mr-1 size-3 md:mr-2 md:size-4" />
            Add...
          </Button>
        )}
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

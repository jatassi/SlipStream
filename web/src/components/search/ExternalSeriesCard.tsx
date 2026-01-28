import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Plus, Library, Clock, Check, CheckCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { PosterImage } from '@/components/media/PosterImage'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { MediaInfoModal } from './MediaInfoModal'
import type { SeriesSearchResult, RequestStatus } from '@/types'

interface RequestInfo {
  id: number
  status: RequestStatus
}

interface ExternalSeriesCardProps {
  series: SeriesSearchResult
  inLibrary?: boolean
  requestInfo?: RequestInfo
  className?: string
}

export function ExternalSeriesCard({ series, inLibrary, requestInfo, className }: ExternalSeriesCardProps) {
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
        'group rounded-lg overflow-hidden bg-card border border-border transition-all hover:border-primary/50 hover:shadow-lg',
        className
      )}
    >
      <div
        className="relative aspect-[2/3] cursor-pointer"
        onClick={() => setInfoOpen(true)}
      >
        <PosterImage
          url={series.posterUrl}
          alt={series.title}
          type="series"
          className="absolute inset-0"
        />

        {/* Status badges */}
        <div className="absolute top-2 left-2 flex flex-col gap-1">
          {inLibrary ? (
            <Badge variant="secondary" className="bg-green-600 text-white text-[10px] md:text-xs px-1.5 md:px-2 py-0.5">
              <Library className="size-2.5 md:size-3 mr-0.5 md:mr-1" />
              In Library
            </Badge>
          ) : requestInfo && (
            isAvailable ? (
              <Badge variant="secondary" className="bg-green-600 text-white text-[10px] md:text-xs px-1.5 md:px-2 py-0.5">
                <CheckCircle className="size-2.5 md:size-3 mr-0.5 md:mr-1" />
                Available
              </Badge>
            ) : isApproved ? (
              <Badge variant="secondary" className="bg-blue-600 text-white text-[10px] md:text-xs px-1.5 md:px-2 py-0.5">
                <Check className="size-2.5 md:size-3 mr-0.5 md:mr-1" />
                Approved
              </Badge>
            ) : isPending && (
              <Badge variant="secondary" className="bg-yellow-600 text-white text-[10px] md:text-xs px-1.5 md:px-2 py-0.5">
                <Clock className="size-2.5 md:size-3 mr-0.5 md:mr-1" />
                Requested
              </Badge>
            )
          )}
        </div>

        {/* Hover overlay */}
        <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity" />
        <div className="absolute inset-x-0 bottom-0 p-3 flex flex-col justify-end opacity-0 group-hover:opacity-100 transition-opacity">
          <h3 className="font-semibold text-white line-clamp-3">{series.title}</h3>
          <div className="flex items-center gap-2 text-sm text-gray-300">
            <span>{series.year || 'Unknown year'}</span>
            {series.network && (
              <Badge variant="secondary" className="text-xs">
                {series.network}
              </Badge>
            )}
          </div>
        </div>
      </div>

      <div className="p-2">
        {inLibrary ? (
          <Button variant="secondary" size="sm" className="w-full text-xs md:text-sm" disabled>
            <Check className="size-3 md:size-4 mr-1 md:mr-2" />
            In Library
          </Button>
        ) : (
          <Button variant="default" size="sm" className="w-full text-xs md:text-sm" onClick={handleAdd}>
            <Plus className="size-3 md:size-4 mr-1 md:mr-2" />
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
        onAction={!inLibrary ? handleAdd : undefined}
        actionLabel="Add..."
        actionIcon={<Plus className="size-3 md:size-4 mr-1 md:mr-2" />}
        disabledLabel="Already Added"
      />
    </div>
  )
}

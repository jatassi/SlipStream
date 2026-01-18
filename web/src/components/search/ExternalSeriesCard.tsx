import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Plus, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { PosterImage } from '@/components/media/PosterImage'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { MediaInfoModal } from './MediaInfoModal'
import type { SeriesSearchResult } from '@/types'

interface ExternalSeriesCardProps {
  series: SeriesSearchResult
  inLibrary?: boolean
  className?: string
}

export function ExternalSeriesCard({ series, inLibrary, className }: ExternalSeriesCardProps) {
  const navigate = useNavigate()
  const [infoOpen, setInfoOpen] = useState(false)

  const handleAdd = (e: React.MouseEvent) => {
    e.stopPropagation()
    navigate({ to: '/series/add', search: { tmdbId: series.tmdbId } })
  }

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
        {inLibrary && (
          <div className="absolute top-2 right-2">
            <Badge variant="secondary" className="bg-green-600 text-white">
              <Check className="size-3 mr-1" />
              In Library
            </Badge>
          </div>
        )}
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
          <Button variant="secondary" size="sm" className="w-full" disabled>
            <Check className="size-4 mr-2" />
            Already Added
          </Button>
        ) : (
          <Button variant="default" size="sm" className="w-full" onClick={handleAdd}>
            <Plus className="size-4 mr-2" />
            Add to Library
          </Button>
        )}
      </div>

      <MediaInfoModal
        open={infoOpen}
        onOpenChange={setInfoOpen}
        media={series}
        mediaType="series"
        inLibrary={inLibrary}
      />
    </div>
  )
}

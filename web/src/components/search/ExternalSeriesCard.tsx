import { useNavigate } from '@tanstack/react-router'
import { Plus } from 'lucide-react'
import { ExternalMediaCard } from './ExternalMediaCard'
import type { SeriesSearchResult } from '@/types'

interface ExternalSeriesCardProps {
  series: SeriesSearchResult
  inLibrary?: boolean
  className?: string
}

export function ExternalSeriesCard({ series, inLibrary, className }: ExternalSeriesCardProps) {
  const navigate = useNavigate()

  const handleAdd = () => {
    navigate({ to: '/series/add', search: { tmdbId: series.tmdbId } })
  }

  return (
    <ExternalMediaCard
      media={series}
      mediaType="series"
      inLibrary={inLibrary}
      className={className}
      onAction={handleAdd}
      actionLabel="Add..."
      actionIcon={<Plus className="size-3 md:size-4 mr-1 md:mr-2" />}
      disabledLabel="Already Added"
    />
  )
}

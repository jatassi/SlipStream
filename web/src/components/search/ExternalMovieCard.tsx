import { useNavigate } from '@tanstack/react-router'
import { Plus } from 'lucide-react'
import { ExternalMediaCard } from './ExternalMediaCard'
import type { MovieSearchResult } from '@/types'

interface ExternalMovieCardProps {
  movie: MovieSearchResult
  inLibrary?: boolean
  className?: string
}

export function ExternalMovieCard({ movie, inLibrary, className }: ExternalMovieCardProps) {
  const navigate = useNavigate()

  const handleAdd = () => {
    navigate({ to: '/movies/add', search: { tmdbId: movie.tmdbId } })
  }

  return (
    <ExternalMediaCard
      media={movie}
      mediaType="movie"
      inLibrary={inLibrary}
      className={className}
      onAction={handleAdd}
      actionLabel="Add..."
      actionIcon={<Plus className="size-3 md:size-4 mr-1 md:mr-2" />}
      disabledLabel="Already Added"
    />
  )
}

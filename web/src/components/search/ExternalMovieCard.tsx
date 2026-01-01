import { useNavigate } from '@tanstack/react-router'
import { Plus, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { PosterImage } from '@/components/media/PosterImage'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { MovieSearchResult } from '@/types'

interface ExternalMovieCardProps {
  movie: MovieSearchResult
  inLibrary?: boolean
  className?: string
}

export function ExternalMovieCard({ movie, inLibrary, className }: ExternalMovieCardProps) {
  const navigate = useNavigate()

  const handleAdd = (e: React.MouseEvent) => {
    e.stopPropagation()
    navigate({ to: '/movies/add', search: { tmdbId: movie.tmdbId } })
  }

  return (
    <div
      className={cn(
        'group rounded-lg overflow-hidden bg-card border border-border transition-all hover:border-primary/50 hover:shadow-lg',
        className
      )}
    >
      <div className="relative aspect-[2/3]">
        <PosterImage
          url={movie.posterUrl}
          alt={movie.title}
          type="movie"
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
        <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 to-transparent p-3">
          <h3 className="font-semibold text-white truncate">{movie.title}</h3>
          <p className="text-sm text-gray-300">{movie.year || 'Unknown year'}</p>
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
    </div>
  )
}

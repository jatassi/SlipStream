import { PosterImage } from '@/components/media/poster-image'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'

import { MediaActionButton } from './media-action-button'
import type { MediaInfoModalProps, MediaInfoState } from './media-info-modal-types'

type MediaInfoHeaderProps = Pick<MediaInfoModalProps, 'media' | 'mediaType' | 'onAction' | 'actionLabel' | 'actionIcon'> & MediaInfoState

export function MediaInfoHeader({
  media,
  mediaType,
  onAction,
  actionLabel = 'Add to Library',
  actionIcon,
  isLoading,
  director,
  creators,
  studio,
  ...actionProps
}: MediaInfoHeaderProps) {
  return (
    <div className="flex gap-4">
      <div className="w-28 shrink-0">
        <PosterImage url={media.posterUrl} alt={media.title} type={mediaType} className="rounded-lg" />
      </div>
      <div className="min-w-0 flex-1 space-y-2">
        <MediaTitle title={media.title} year={media.year} />
        <MetadataTags media={media} contentRating={actionProps.extendedData?.contentRating} />
        {isLoading ? (
          <CreditsSkeleton />
        ) : (
          <CreditsInfo mediaType={mediaType} director={director} creators={creators} studio={studio} />
        )}
        <MediaActionButton
          onAction={onAction}
          actionLabel={actionLabel}
          actionIcon={actionIcon}
          {...actionProps}
        />
      </div>
    </div>
  )
}

function MediaTitle({ title, year }: { title: string; year?: number }) {
  return (
    <h2 className="text-xl font-bold">
      {title}
      {year ? <span className="text-muted-foreground ml-2 font-normal">({year})</span> : null}
    </h2>
  )
}

function MetadataTags({ media, contentRating }: { media: { runtime?: number; genres?: string[] }; contentRating?: string }) {
  return (
    <div className="text-muted-foreground flex flex-wrap items-center gap-2 text-sm">
      {contentRating ? <Badge variant="outline">{contentRating}</Badge> : null}
      {media.runtime ? <span>{formatRuntime(media.runtime)}</span> : null}
      {media.genres?.slice(0, 3).map((genre) => (
        <Badge key={genre} variant="secondary">{genre}</Badge>
      ))}
    </div>
  )
}

function CreditsSkeleton() {
  return (
    <div className="space-y-2">
      <Skeleton className="h-4 w-32" />
      <Skeleton className="h-4 w-24" />
    </div>
  )
}

function CreditsInfo({
  mediaType,
  director,
  creators,
  studio,
}: {
  mediaType: 'movie' | 'series'
  director: string | undefined
  creators: { name: string }[] | undefined
  studio: string | undefined
}) {
  const items: { label: string; value: string }[] = []
  if (mediaType === 'movie' && director) {
    items.push({ label: 'Director', value: director })
  }
  if (mediaType === 'series' && creators && creators.length > 0) {
    items.push({ label: 'Created by', value: creators.map((c) => c.name).join(', ') })
  }
  if (studio) {
    items.push({ label: 'Studio', value: studio })
  }
  if (items.length === 0) {
    return null
  }
  return (
    <div className="space-y-1 text-sm">
      {items.map((item) => (
        <p key={item.label}>
          <span className="text-muted-foreground">{item.label}:</span> {item.value}
        </p>
      ))}
    </div>
  )
}

function formatRuntime(minutes?: number) {
  if (!minutes) {
    return null
  }
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  return hours > 0 ? `${hours}h ${mins}m` : `${mins}m`
}

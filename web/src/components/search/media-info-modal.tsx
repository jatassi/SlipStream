import { Plus } from 'lucide-react'

import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'

import { CastList } from './cast-list'
import { MediaInfoHeader } from './media-info-header'
import type { MediaInfoModalProps } from './media-info-modal-types'
import { RatingsDisplay } from './ratings-display'
import { SeasonsList } from './seasons-list'
import { useMediaInfoModal } from './use-media-info-modal'

export type { MediaInfoModalProps, MediaInfoState } from './media-info-modal-types'

export function MediaInfoModal({
  open,
  onOpenChange,
  media,
  mediaType,
  inLibrary,
  onAction,
  actionLabel = 'Add to Library',
  actionIcon = <Plus className="mr-2 size-4" />,
}: MediaInfoModalProps) {
  const state = useMediaInfoModal({ open, onOpenChange, media, mediaType, inLibrary, onAction })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[85vh] flex-col overflow-hidden p-0 sm:max-w-2xl">
        <DialogTitle className="sr-only">{media.title}</DialogTitle>
        <ScrollArea className="min-h-0 flex-1">
          <div className="space-y-4 p-4">
            <MediaInfoHeader
              media={media}
              mediaType={mediaType}
              onAction={onAction}
              actionLabel={actionLabel}
              actionIcon={actionIcon}
              {...state}
            />

            {media.overview ? (
              <p className="text-muted-foreground text-sm leading-relaxed">{media.overview}</p>
            ) : null}

            <RatingsOrSkeleton
              isLoading={state.isLoading}
              ratings={state.extendedData?.ratings}
            />

            <CastOrSkeleton
              isLoading={state.isLoading}
              cast={state.extendedData?.credits?.cast}
            />

            {mediaType === 'series' && state.seasons && state.seasons.length > 0 ? (
              <SeasonsList seasons={state.seasons} />
            ) : null}
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}

function RatingsOrSkeleton({
  isLoading,
  ratings,
}: {
  isLoading: boolean
  ratings: Parameters<typeof RatingsDisplay>[0]['ratings'] | undefined
}) {
  if (isLoading) {
    return (
      <div className="flex gap-4">
        <Skeleton className="h-8 w-20" />
        <Skeleton className="h-8 w-20" />
        <Skeleton className="h-8 w-20" />
      </div>
    )
  }
  if (!ratings) {
    return null
  }
  return <RatingsDisplay ratings={ratings} />
}

function CastOrSkeleton({
  isLoading,
  cast,
}: {
  isLoading: boolean
  cast: { id: number; name: string; photoUrl?: string; role?: string }[] | undefined
}) {
  if (isLoading) {
    return (
      <div>
        <h3 className="mb-2 text-sm font-semibold">Cast</h3>
        <div className="flex gap-3 overflow-x-auto pb-2">
          {Array.from({ length: 6 }, (_, i) => i).map((i) => (
            <div key={i} className="flex shrink-0 flex-col items-center gap-1">
              <Skeleton className="size-16 rounded-full" />
              <Skeleton className="h-3 w-14" />
            </div>
          ))}
        </div>
      </div>
    )
  }
  if (!cast || cast.length === 0) {
    return null
  }
  return <CastList cast={cast} />
}

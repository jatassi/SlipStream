import type { ReactNode } from 'react'

import { Check } from 'lucide-react'

import { PosterImage } from '@/components/media/poster-image'
import { Button } from '@/components/ui/button'
import type { QualityProfile, RootFolder } from '@/types'

import { FolderSelect, ProfileSelect } from './media-configure-fields'

export type MediaPreviewProps = {
  title: string
  year: number | undefined
  overview: string | undefined
  posterUrl: string | undefined
  type: 'movie' | 'series'
  subtitle?: string
}

export function MediaPreview({ title, year, overview, posterUrl, type, subtitle }: MediaPreviewProps) {
  return (
    <div className="flex gap-4">
      <PosterImage
        url={posterUrl}
        alt={title}
        type={type}
        className="h-32 w-[86px] shrink-0 rounded-[10px] shadow-[0_6px_16px_rgba(0,0,0,0.35)]"
      />
      <div className="min-w-0">
        <h2 className="text-title font-semibold">{title}</h2>
        <p className="text-footnote text-muted-foreground nums">
          {year ?? 'Unknown year'}
          {subtitle === undefined ? null : ` · ${subtitle}`}
        </p>
        {!!overview && (
          <p className="text-footnote text-muted-foreground mt-2 line-clamp-3">{overview}</p>
        )}
      </div>
    </div>
  )
}

export type AddMediaConfigureProps = {
  preview: ReactNode
  rootFolders: RootFolder[] | undefined
  qualityProfiles: QualityProfile[] | undefined
  rootFolderId: string
  qualityProfileId: string
  onFolderChange: (v: string) => void
  onProfileChange: (v: string) => void
  children: ReactNode
}

export function AddMediaConfigure({
  preview,
  rootFolders,
  qualityProfiles,
  rootFolderId,
  qualityProfileId,
  onFolderChange,
  onProfileChange,
  children,
}: AddMediaConfigureProps) {
  return (
    <div className="space-y-5 pb-2">
      {preview}
      <div className="space-y-4">
        <FolderSelect
          rootFolderId={rootFolderId}
          rootFolders={rootFolders}
          onChange={onFolderChange}
        />
        <ProfileSelect
          qualityProfileId={qualityProfileId}
          qualityProfiles={qualityProfiles}
          onChange={onProfileChange}
        />
        {children}
      </div>
    </div>
  )
}

export type AddMediaActionsProps = {
  rootFolderId: string
  qualityProfileId: string
  isPending: boolean
  onCancel: () => void
  onAdd: () => void
  addLabel: string
}

export function AddMediaActions({
  rootFolderId,
  qualityProfileId,
  isPending,
  onCancel,
  onAdd,
  addLabel,
}: AddMediaActionsProps) {
  return (
    <div className="flex w-full items-center gap-2">
      <Button variant="outline" className="min-h-tap" onClick={onCancel}>
        Cancel
      </Button>
      <Button
        className="min-h-tap flex-1"
        onClick={onAdd}
        disabled={!rootFolderId || !qualityProfileId || isPending}
      >
        <Check className="mr-2 size-4" />
        {addLabel}
      </Button>
    </div>
  )
}

import type { ReactNode } from 'react'

import { PosterImage } from '@/components/media/poster-image'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { QualityProfile, RootFolder } from '@/types'

import {
  FolderSelect,
  FormActions,
  ProfileSelect,
} from './media-configure-fields'

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
    <Card>
      <CardContent className="flex gap-4 p-4">
        <PosterImage
          url={posterUrl}
          alt={title}
          type={type}
          className="h-36 w-24 shrink-0 rounded"
        />
        <div>
          <h2 className="text-xl font-semibold">{title}</h2>
          <p className="text-muted-foreground">
            {year ?? 'Unknown year'}
            {subtitle ? ` - ${subtitle}` : null}
          </p>
          {!!overview && (
            <p className="text-muted-foreground mt-2 line-clamp-3 text-sm">
              {overview}
            </p>
          )}
        </div>
      </CardContent>
    </Card>
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
  isPending: boolean
  onBack: () => void
  onAdd: () => void
  addLabel: string
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
  isPending,
  onBack,
  onAdd,
  addLabel,
  children,
}: AddMediaConfigureProps) {
  return (
    <div className="max-w-2xl space-y-6">
      {preview}
      <Card>
        <CardHeader>
          <CardTitle>Configuration</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
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
        </CardContent>
      </Card>
      <FormActions
        rootFolderId={rootFolderId}
        qualityProfileId={qualityProfileId}
        isPending={isPending}
        onBack={onBack}
        onAdd={onAdd}
        addLabel={addLabel}
      />
    </div>
  )
}

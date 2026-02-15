import { Check } from 'lucide-react'

import { PosterImage } from '@/components/media/poster-image'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { MovieSearchResult, QualityProfile, RootFolder } from '@/types'

type AddMovieConfigureProps = {
  selectedMovie: MovieSearchResult
  rootFolderId: string
  setRootFolderId: (v: string) => void
  rootFolders: RootFolder[] | undefined
  qualityProfileId: string
  setQualityProfileId: (v: string) => void
  qualityProfiles: QualityProfile[] | undefined
  monitored: boolean
  setMonitored: (v: boolean) => void
  searchOnAdd: boolean | undefined
  setSearchOnAdd: (v: boolean) => void
  isPending: boolean
  handleBack: () => void
  handleAdd: () => void
}

export function AddMovieConfigure(props: AddMovieConfigureProps) {
  return (
    <div className="max-w-2xl space-y-6">
      <MoviePreview movie={props.selectedMovie} />
      <ConfigurationForm {...props} />
      <FormActions
        rootFolderId={props.rootFolderId}
        qualityProfileId={props.qualityProfileId}
        isPending={props.isPending}
        onBack={props.handleBack}
        onAdd={props.handleAdd}
      />
    </div>
  )
}

function MoviePreview({ movie }: { movie: MovieSearchResult }) {
  return (
    <Card>
      <CardContent className="flex gap-4 p-4">
        <PosterImage
          url={movie.posterUrl}
          alt={movie.title}
          type="movie"
          className="h-36 w-24 shrink-0 rounded"
        />
        <div>
          <h2 className="text-xl font-semibold">{movie.title}</h2>
          <p className="text-muted-foreground">{movie.year ?? 'Unknown year'}</p>
          {movie.overview ? (
            <p className="text-muted-foreground mt-2 line-clamp-3 text-sm">
              {movie.overview}
            </p>
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}

type FormActionsProps = {
  rootFolderId: string
  qualityProfileId: string
  isPending: boolean
  onBack: () => void
  onAdd: () => void
}

function FormActions({ rootFolderId, qualityProfileId, isPending, onBack, onAdd }: FormActionsProps) {
  return (
    <div className="flex justify-end gap-2">
      <Button variant="outline" onClick={onBack}>
        Back
      </Button>
      <Button onClick={onAdd} disabled={!rootFolderId || !qualityProfileId || isPending}>
        <Check className="mr-2 size-4" />
        Add Movie
      </Button>
    </div>
  )
}

function ConfigurationForm(props: AddMovieConfigureProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Configuration</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <FolderSelect
          rootFolderId={props.rootFolderId}
          rootFolders={props.rootFolders}
          onChange={props.setRootFolderId}
        />
        <ProfileSelect
          qualityProfileId={props.qualityProfileId}
          qualityProfiles={props.qualityProfiles}
          onChange={props.setQualityProfileId}
        />
        <ToggleField
          label="Monitored"
          description="Automatically search for and download releases"
          checked={props.monitored}
          onChange={props.setMonitored}
        />
        <ToggleField
          label="Search on Add"
          description="Start searching for releases immediately"
          checked={props.searchOnAdd ?? false}
          onChange={props.setSearchOnAdd}
        />
      </CardContent>
    </Card>
  )
}

type FolderSelectProps = {
  rootFolderId: string
  rootFolders: RootFolder[] | undefined
  onChange: (v: string) => void
}

function FolderSelect({ rootFolderId, rootFolders, onChange }: FolderSelectProps) {
  const label =
    rootFolders?.find((f) => f.id === Number.parseInt(rootFolderId))?.name ??
    'Select a root folder'

  return (
    <div className="space-y-2">
      <Label htmlFor="rootFolder">Root Folder *</Label>
      <Select value={rootFolderId} onValueChange={(v) => { if (v) { onChange(v) } }}>
        <SelectTrigger>{label}</SelectTrigger>
        <SelectContent>
          {rootFolders?.map((folder) => (
            <SelectItem key={folder.id} value={String(folder.id)}>
              {folder.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

type ProfileSelectProps = {
  qualityProfileId: string
  qualityProfiles: QualityProfile[] | undefined
  onChange: (v: string) => void
}

function ProfileSelect({ qualityProfileId, qualityProfiles, onChange }: ProfileSelectProps) {
  const label =
    qualityProfiles?.find((p) => p.id === Number.parseInt(qualityProfileId))?.name ??
    'Select a quality profile'

  return (
    <div className="space-y-2">
      <Label htmlFor="qualityProfile">Quality Profile *</Label>
      <Select value={qualityProfileId} onValueChange={(v) => { if (v) { onChange(v) } }}>
        <SelectTrigger>{label}</SelectTrigger>
        <SelectContent>
          {qualityProfiles?.map((profile) => (
            <SelectItem key={profile.id} value={String(profile.id)}>
              {profile.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

type ToggleFieldProps = {
  label: string
  description: string
  checked: boolean
  onChange: (checked: boolean) => void
}

function ToggleField({ label, description, checked, onChange }: ToggleFieldProps) {
  return (
    <div className="flex items-center justify-between">
      <div className="space-y-0.5">
        <Label>{label}</Label>
        <p className="text-muted-foreground text-sm">{description}</p>
      </div>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  )
}

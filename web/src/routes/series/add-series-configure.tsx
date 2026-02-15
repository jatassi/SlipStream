import { Check } from 'lucide-react'

import { PosterImage } from '@/components/media/poster-image'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { QualityProfile, RootFolder, SeriesMonitorOnAdd, SeriesSearchOnAdd, SeriesSearchResult } from '@/types'

import { MONITOR_LABELS, SEARCH_ON_ADD_LABELS } from './add-series-constants'
import type { AddSeriesState } from './use-add-series'

type AddSeriesConfigureProps = Pick<
  AddSeriesState,
  | 'selectedSeries'
  | 'rootFolderId'
  | 'setRootFolderId'
  | 'qualityProfileId'
  | 'setQualityProfileId'
  | 'monitorOnAdd'
  | 'setMonitorOnAdd'
  | 'searchOnAdd'
  | 'setSearchOnAdd'
  | 'seasonFolder'
  | 'setSeasonFolder'
  | 'includeSpecials'
  | 'setIncludeSpecials'
  | 'isPending'
  | 'handleBack'
  | 'handleAdd'
> & {
  rootFolders: RootFolder[] | undefined
  qualityProfiles: QualityProfile[] | undefined
}

export function AddSeriesConfigure(props: AddSeriesConfigureProps) {
  if (!props.selectedSeries) {
    return null
  }

  return (
    <div className="max-w-2xl space-y-6">
      <SeriesPreview series={props.selectedSeries} />
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

function SeriesPreview({ series }: { series: SeriesSearchResult }) {
  return (
    <Card>
      <CardContent className="flex gap-4 p-4">
        <PosterImage
          url={series.posterUrl}
          alt={series.title}
          type="series"
          className="h-36 w-24 shrink-0 rounded"
        />
        <div>
          <h2 className="text-xl font-semibold">{series.title}</h2>
          <p className="text-muted-foreground">
            {series.year ?? 'Unknown year'}
            {series.network ? ` - ${series.network}` : null}
          </p>
          {series.overview ? (
            <p className="text-muted-foreground mt-2 line-clamp-3 text-sm">
              {series.overview}
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
        Add Series
      </Button>
    </div>
  )
}

function ConfigurationForm(props: AddSeriesConfigureProps) {
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
        <MonitorSelect value={props.monitorOnAdd} onChange={(v) => props.setMonitorOnAdd(v as SeriesMonitorOnAdd)} />
        <SearchOnAddSelect value={props.searchOnAdd} onChange={(v) => props.setSearchOnAdd(v as SeriesSearchOnAdd)} />
        <ToggleField
          label="Season Folder"
          description="Organize episodes into season folders"
          checked={props.seasonFolder}
          onChange={props.setSeasonFolder}
        />
        <ToggleField
          label="Include Specials"
          description="Monitor and search for special episodes (Season 0)"
          checked={props.includeSpecials ?? false}
          onChange={props.setIncludeSpecials}
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

type MonitorSelectProps = {
  value: string | undefined
  onChange: (v: string) => void
}

function MonitorSelect({ value, onChange }: MonitorSelectProps) {
  const resolved = value ?? 'future'
  return (
    <div className="space-y-2">
      <Label>Monitor</Label>
      <Select value={resolved} onValueChange={(v) => { if (v) { onChange(v) } }}>
        <SelectTrigger>{MONITOR_LABELS[resolved as keyof typeof MONITOR_LABELS]}</SelectTrigger>
        <SelectContent>
          {Object.entries(MONITOR_LABELS).map(([k, label]) => (
            <SelectItem key={k} value={k}>{label}</SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="text-muted-foreground text-sm">
        Which episodes should be monitored for automatic downloads
      </p>
    </div>
  )
}

type SearchOnAddSelectProps = {
  value: string | undefined
  onChange: (v: string) => void
}

function SearchOnAddSelect({ value, onChange }: SearchOnAddSelectProps) {
  const resolved = value ?? 'no'
  return (
    <div className="space-y-2">
      <Label>Search on Add</Label>
      <Select value={resolved} onValueChange={(v) => { if (v) { onChange(v) } }}>
        <SelectTrigger>{SEARCH_ON_ADD_LABELS[resolved as keyof typeof SEARCH_ON_ADD_LABELS]}</SelectTrigger>
        <SelectContent>
          {Object.entries(SEARCH_ON_ADD_LABELS).map(([k, label]) => (
            <SelectItem key={k} value={k}>{label}</SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="text-muted-foreground text-sm">
        Start searching for releases immediately after adding
      </p>
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

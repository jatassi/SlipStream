import { useState } from 'react'

import { Pencil } from 'lucide-react'

import { Card, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { RootFolder, Slot } from '@/types'

type SlotCardProps = {
  slot: Slot
  profiles: { id: number; name: string }[]
  usedProfileIds: number[]
  movieRootFolders: RootFolder[]
  tvRootFolders: RootFolder[]
  onEnabledChange: (enabled: boolean) => void
  onNameChange: (name: string) => void
  onProfileChange: (profileId: string) => void
  onRootFolderChange: (mediaType: 'movie' | 'tv', rootFolderId: string) => void
  isUpdating: boolean
  showToggle?: boolean
}

function SlotToggle({
  slot,
  onEnabledChange,
  isUpdating,
}: {
  slot: Slot
  onEnabledChange: (enabled: boolean) => void
  isUpdating: boolean
}) {
  if (slot.qualityProfileId === null) {
    return (
      <Tooltip>
        <TooltipTrigger>
          <Switch checked={slot.enabled} disabled className="shrink-0" />
        </TooltipTrigger>
        <TooltipContent side="left">
          <p>Select a quality profile first</p>
        </TooltipContent>
      </Tooltip>
    )
  }

  return (
    <Switch
      checked={slot.enabled}
      onCheckedChange={onEnabledChange}
      disabled={isUpdating}
      className="shrink-0"
    />
  )
}

function SlotNameInput({
  slot,
  onNameChange,
}: {
  slot: Slot
  onNameChange: (name: string) => void
}) {
  const [editingName, setEditingName] = useState(false)
  const [tempName, setTempName] = useState(slot.name)

  const handleNameSubmit = () => {
    if (tempName.trim() && tempName !== slot.name) {
      onNameChange(tempName.trim())
    }
    setEditingName(false)
  }

  return (
    <div
      className={`group relative flex items-center rounded-md border transition-colors ${
        editingName
          ? 'border-primary bg-background'
          : 'hover:border-muted-foreground/25 hover:bg-muted/50 border-transparent'
      }`}
    >
      <Input
        value={tempName}
        onChange={(e) => setTempName(e.target.value)}
        onFocus={() => setEditingName(true)}
        onBlur={handleNameSubmit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            handleNameSubmit()
            e.currentTarget.blur()
          }
          if (e.key === 'Escape') {
            setTempName(slot.name)
            setEditingName(false)
            e.currentTarget.blur()
          }
        }}
        className="h-8 border-0 bg-transparent pr-8 text-base font-semibold tracking-tight focus-visible:ring-0 focus-visible:ring-offset-0"
      />
      <Pencil
        className={`text-muted-foreground absolute right-2 size-3.5 transition-opacity ${
          editingName ? 'opacity-0' : 'opacity-0 group-hover:opacity-100'
        }`}
      />
    </div>
  )
}

function RootFolderSelect({
  slotId,
  mediaType,
  selectedId,
  folders,
  onRootFolderChange,
  isUpdating,
}: {
  slotId: number
  mediaType: 'movie' | 'tv'
  selectedId: number | null
  folders: RootFolder[]
  onRootFolderChange: (mediaType: 'movie' | 'tv', rootFolderId: string) => void
  isUpdating: boolean
}) {
  const label = mediaType === 'movie' ? 'Movie Root Folder' : 'TV Root Folder'
  const htmlId = `slot-${slotId}-${mediaType}-root`

  return (
    <div className="space-y-2">
      <Label htmlFor={htmlId}>{label}</Label>
      <Select
        value={selectedId?.toString() ?? 'none'}
        onValueChange={(v) => v && onRootFolderChange(mediaType, v)}
        disabled={isUpdating}
      >
        <SelectTrigger id={htmlId}>
          {folders.find((f) => f.id === selectedId)?.name ?? 'Use media default'}
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="none">Use media default</SelectItem>
          {folders.map((folder) => (
            <SelectItem key={folder.id} value={folder.id.toString()}>
              {folder.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function ProfileSelect({
  slotId,
  selectedProfileId,
  profileName,
  availableProfiles,
  onProfileChange,
  isUpdating,
}: {
  slotId: number
  selectedProfileId: number | null
  profileName: string | undefined
  availableProfiles: { id: number; name: string }[]
  onProfileChange: (profileId: string) => void
  isUpdating: boolean
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor={`slot-${slotId}-profile`}>Quality Profile</Label>
      <Select
        value={selectedProfileId?.toString() ?? 'none'}
        onValueChange={(v) => v && onProfileChange(v)}
        disabled={isUpdating}
      >
        <SelectTrigger id={`slot-${slotId}-profile`}>
          {profileName ?? 'Select profile...'}
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="none">None</SelectItem>
          {availableProfiles.map((profile) => (
            <SelectItem key={profile.id} value={profile.id.toString()}>
              {profile.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function SlotCardHeader({
  slot,
  showToggle,
  onEnabledChange,
  onNameChange,
  isUpdating,
}: {
  slot: Slot
  showToggle: boolean
  onEnabledChange: (enabled: boolean) => void
  onNameChange: (name: string) => void
  isUpdating: boolean
}) {
  return (
    <CardHeader className="pb-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex shrink-0 items-center gap-2">
          <div className="bg-primary/10 text-primary flex h-8 w-8 items-center justify-center rounded-full font-semibold">
            {slot.slotNumber}
          </div>
        </div>
        <div className="min-w-0 flex-1">
          <SlotNameInput slot={slot} onNameChange={onNameChange} />
        </div>
        {showToggle ? (
          <SlotToggle slot={slot} onEnabledChange={onEnabledChange} isUpdating={isUpdating} />
        ) : null}
      </div>
      <CardDescription>
        {slot.enabled ? 'Active' : 'Disabled'}
        {slot.fileCount !== undefined && slot.fileCount > 0 && ` - ${slot.fileCount} files`}
      </CardDescription>
    </CardHeader>
  )
}

export function SlotCard(props: SlotCardProps) {
  const { slot, isUpdating } = props
  const availableProfiles = props.profiles.filter((p) => !props.usedProfileIds.includes(p.id))

  return (
    <Card className={slot.enabled ? 'ring-primary/50 ring-2' : ''}>
      <SlotCardHeader
        slot={slot}
        showToggle={props.showToggle ?? false}
        onEnabledChange={props.onEnabledChange}
        onNameChange={props.onNameChange}
        isUpdating={isUpdating}
      />
      <CardContent>
        <div className="space-y-4">
          <ProfileSelect
            slotId={slot.id}
            selectedProfileId={slot.qualityProfileId}
            profileName={slot.qualityProfile?.name}
            availableProfiles={availableProfiles}
            onProfileChange={props.onProfileChange}
            isUpdating={isUpdating}
          />
          <RootFolderSelect
            slotId={slot.id}
            mediaType="movie"
            selectedId={slot.movieRootFolderId}
            folders={props.movieRootFolders}
            onRootFolderChange={props.onRootFolderChange}
            isUpdating={isUpdating}
          />
          <RootFolderSelect
            slotId={slot.id}
            mediaType="tv"
            selectedId={slot.tvRootFolderId}
            folders={props.tvRootFolders}
            onRootFolderChange={props.onRootFolderChange}
            isUpdating={isUpdating}
          />
        </div>
      </CardContent>
    </Card>
  )
}

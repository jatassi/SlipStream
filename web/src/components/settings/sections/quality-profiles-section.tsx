import { useMemo, useState } from 'react'

import { Edit, Plus, Sliders, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

import { EmptyState } from '@/components/data/empty-state'
import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { QualityProfileDialog } from '@/components/qualityprofiles'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { useDeleteQualityProfile, useQualityProfiles } from '@/hooks'
import { withToast } from '@/lib/with-toast'
import { getEnabledModules, getModule } from '@/modules'
import type { QualityProfile } from '@/types'
import { PREDEFINED_QUALITIES } from '@/types'

const getCutoffName = (cutoffId: number) => {
  const quality = PREDEFINED_QUALITIES.find((q) => q.id === cutoffId)
  return quality?.name ?? 'Unknown'
}

function QualityBadgeList({ profile }: { profile: QualityProfile }) {
  const allowedQualities = profile.items.filter((item) => item.allowed).map((item) => item.quality.name)
  return (
    <div className="flex gap-1 overflow-hidden">
      {allowedQualities.slice(0, 4).map((quality) => (
        <Badge key={quality} variant="secondary" className="shrink-0">{quality}</Badge>
      ))}
      {allowedQualities.length > 4 && (
        <Badge variant="outline" className="shrink-0">+{allowedQualities.length - 4} more</Badge>
      )}
    </div>
  )
}

const BADGE_CLASSES: Record<string, string> = {
  movie: 'bg-movie-500/10 text-movie-500',
  tv: 'bg-tv-500/10 text-tv-500',
}

function ModuleTypeBadge({ moduleType }: { moduleType: string }) {
  const mod = getModule(moduleType)
  const label = mod?.singularName ?? moduleType
  const className = BADGE_CLASSES[moduleType] ?? ''
  return <Badge variant="secondary" className={`text-xs ${className}`}>{label}</Badge>
}

function ProfileCard({
  profile,
  onEdit,
  onDelete,
}: {
  profile: QualityProfile
  onEdit: (profile: QualityProfile) => void
  onDelete: (id: number) => void
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between">
        <div>
          <div className="flex items-center gap-2">
            <CardTitle className="text-lg">{profile.name}</CardTitle>
            <ModuleTypeBadge moduleType={profile.moduleType} />
            {profile.allowAutoApprove ? <Badge variant="outline" className="text-xs">Auto-Approve</Badge> : null}
          </div>
          <CardDescription>
            {profile.upgradesEnabled ? (
              <>Cutoff: {getCutoffName(profile.cutoff)}</>
            ) : (
              <span className="text-muted-foreground/70">Upgrades disabled</span>
            )}
          </CardDescription>
        </div>
        <div className="flex gap-1">
          <Button variant="ghost" size="icon" aria-label="Edit" onClick={() => onEdit(profile)}>
            <Edit className="size-4" />
          </Button>
          <ConfirmDialog
            trigger={<Button variant="ghost" size="icon" aria-label="Delete"><Trash2 className="size-4" /></Button>}
            title="Delete profile"
            description={`Are you sure you want to delete "${profile.name}"?`}
            confirmLabel="Delete"
            variant="destructive"
            onConfirm={() => onDelete(profile.id)}
          />
        </div>
      </CardHeader>
      <CardContent>
        <QualityBadgeList profile={profile} />
      </CardContent>
    </Card>
  )
}

function useProfilesByModule(profiles: QualityProfile[] | undefined) {
  return useMemo(() => {
    const grouped = new Map<string, QualityProfile[]>()
    for (const profile of profiles ?? []) {
      const list = grouped.get(profile.moduleType) ?? []
      list.push(profile)
      grouped.set(profile.moduleType, list)
    }
    return grouped
  }, [profiles])
}

function GroupedProfileList({
  profilesByModule,
  onEdit,
  onDelete,
}: {
  profilesByModule: Map<string, QualityProfile[]>
  onEdit: (profile: QualityProfile) => void
  onDelete: (id: number) => void
}) {
  return (
    <div className="space-y-6">
      {getEnabledModules().map((mod) => {
        const moduleProfiles = profilesByModule.get(mod.id) ?? []
        if (moduleProfiles.length === 0) {
          return null
        }
        return (
          <div key={mod.id}>
            <h3 className="text-sm font-medium text-muted-foreground mb-3">{mod.name} Profiles</h3>
            <div className="grid gap-4 md:grid-cols-2">
              {moduleProfiles.map((profile) => (
                <ProfileCard key={profile.id} profile={profile} onEdit={onEdit} onDelete={onDelete} />
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}

export function QualityProfilesSection() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingProfile, setEditingProfile] = useState<QualityProfile | null>(null)

  const { data: profiles, isLoading, isError, refetch } = useQualityProfiles()
  const deleteMutation = useDeleteQualityProfile()
  const profilesByModule = useProfilesByModule(profiles)

  const handleAddProfile = () => { setEditingProfile(null); setDialogOpen(true) }
  const handleEditProfile = (profile: QualityProfile) => { setEditingProfile(profile); setDialogOpen(true) }

  const handleDelete = withToast(async (id: number) => {
    await deleteMutation.mutateAsync(id)
    toast.success('Profile deleted')
  })

  if (isLoading) {
    return <LoadingState variant="list" count={3} />
  }
  if (isError) {
    return <ErrorState onRetry={refetch} />
  }
  if (!profiles?.length) {
    return (
      <>
        <EmptyState
          icon={<Sliders className="size-8" />}
          title="No quality profiles"
          description="Create a quality profile to get started"
          action={{ label: 'Add Profile', onClick: handleAddProfile }}
        />
        <QualityProfileDialog open={dialogOpen} onOpenChange={setDialogOpen} profile={editingProfile} />
      </>
    )
  }

  return (
    <>
      <GroupedProfileList profilesByModule={profilesByModule} onEdit={handleEditProfile} onDelete={handleDelete} />
      <Button variant="outline" size="sm" className="mt-4" onClick={handleAddProfile}>
        <Plus className="mr-2 size-4" />
        Add Quality Profile
      </Button>
      <QualityProfileDialog open={dialogOpen} onOpenChange={setDialogOpen} profile={editingProfile} />
    </>
  )
}

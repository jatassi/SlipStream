import { useState } from 'react'
import { Plus, Edit, Trash2, Sliders } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { QualityProfileDialog } from '@/components/qualityprofiles'
import { useQualityProfiles, useDeleteQualityProfile } from '@/hooks'
import { PREDEFINED_QUALITIES } from '@/types'
import type { QualityProfile } from '@/types'
import { toast } from 'sonner'

export function QualityProfilesPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingProfile, setEditingProfile] = useState<QualityProfile | null>(null)

  const { data: profiles, isLoading, isError, refetch } = useQualityProfiles()
  const deleteMutation = useDeleteQualityProfile()

  const handleAddProfile = () => {
    setEditingProfile(null)
    setDialogOpen(true)
  }

  const handleEditProfile = (profile: QualityProfile) => {
    setEditingProfile(profile)
    setDialogOpen(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
      toast.success('Profile deleted')
    } catch {
      toast.error('Failed to delete profile')
    }
  }

  const getCutoffName = (cutoffId: number) => {
    const quality = PREDEFINED_QUALITIES.find((q) => q.id === cutoffId)
    return quality?.name || 'Unknown'
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Quality Profiles" />
        <LoadingState variant="list" count={3} />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Quality Profiles" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Quality Profiles"
        description="Configure quality preferences for downloads"
        breadcrumbs={[
          { label: 'Settings', href: '/settings' },
          { label: 'Quality Profiles' },
        ]}
        actions={
          <Button onClick={handleAddProfile}>
            <Plus className="size-4 mr-2" />
            Add Profile
          </Button>
        }
      />

      {!profiles?.length ? (
        <EmptyState
          icon={<Sliders className="size-8" />}
          title="No quality profiles"
          description="Create a quality profile to get started"
          action={{ label: 'Add Profile', onClick: handleAddProfile }}
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {profiles.map((profile) => {
            const allowedQualities = profile.items
              .filter((item) => item.allowed)
              .map((item) => item.quality.name)

            return (
              <Card key={profile.id}>
                <CardHeader className="flex flex-row items-start justify-between">
                  <div>
                    <CardTitle className="text-lg">{profile.name}</CardTitle>
                    <CardDescription>
                      {profile.upgradesEnabled ? (
                        <>Cutoff: {getCutoffName(profile.cutoff)}</>
                      ) : (
                        <span className="text-muted-foreground/70">Upgrades disabled</span>
                      )}
                    </CardDescription>
                  </div>
                  <div className="flex gap-1">
                    <Button variant="ghost" size="icon" onClick={() => handleEditProfile(profile)}>
                      <Edit className="size-4" />
                    </Button>
                    <ConfirmDialog
                      trigger={
                        <Button variant="ghost" size="icon">
                          <Trash2 className="size-4" />
                        </Button>
                      }
                      title="Delete profile"
                      description={`Are you sure you want to delete "${profile.name}"?`}
                      confirmLabel="Delete"
                      variant="destructive"
                      onConfirm={() => handleDelete(profile.id)}
                    />
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-wrap gap-1">
                    {allowedQualities.slice(0, 6).map((quality) => (
                      <Badge key={quality} variant="secondary">
                        {quality}
                      </Badge>
                    ))}
                    {allowedQualities.length > 6 && (
                      <Badge variant="outline">
                        +{allowedQualities.length - 6} more
                      </Badge>
                    )}
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}

      <QualityProfileDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        profile={editingProfile}
      />
    </div>
  )
}

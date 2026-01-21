import { useState } from 'react'
import { Edit, Trash2, Sliders } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { QualityProfileDialog } from '@/components/qualityprofiles'
import { ListSection } from '@/components/settings/ListSection'
import { useQualityProfiles, useDeleteQualityProfile } from '@/hooks'
import { PREDEFINED_QUALITIES } from '@/types'
import type { QualityProfile } from '@/types'
import { toast } from 'sonner'

export function QualityProfilesSection() {
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

  const renderProfile = (profile: QualityProfile) => {
    const allowedQualities = profile.items
      .filter((item) => item.allowed)
      .map((item) => item.quality.name)

    return (
      <Card>
        <CardHeader className="flex flex-row items-start justify-between">
          <div>
            <div className="flex items-center gap-2">
              <CardTitle className="text-lg">{profile.name}</CardTitle>
              {profile.allowAutoApprove && (
                <Badge variant="outline" className="text-xs">Auto-Approve</Badge>
              )}
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
          <div className="flex gap-1 overflow-hidden">
            {allowedQualities.slice(0, 4).map((quality) => (
              <Badge key={quality} variant="secondary" className="shrink-0">
                {quality}
              </Badge>
            ))}
            {allowedQualities.length > 4 && (
              <Badge variant="outline" className="shrink-0">
                +{allowedQualities.length - 4} more
              </Badge>
            )}
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <ListSection
        data={profiles}
        isLoading={isLoading}
        isError={isError}
        refetch={refetch}
        emptyIcon={<Sliders className="size-8" />}
        emptyTitle="No quality profiles"
        emptyDescription="Create a quality profile to get started"
        emptyAction={{ label: 'Add Profile', onClick: handleAddProfile }}
        renderItem={renderProfile}
        gridCols={2}
        keyExtractor={(profile) => profile.id}
        addPlaceholder={{ label: 'Add Quality Profile', onClick: handleAddProfile }}
      />

      <QualityProfileDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        profile={editingProfile}
      />
    </>
  )
}

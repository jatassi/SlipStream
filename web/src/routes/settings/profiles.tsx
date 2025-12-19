import { Plus, Edit, Trash2, Sliders } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { useQualityProfiles, useDeleteQualityProfile } from '@/hooks'
import { PREDEFINED_QUALITIES } from '@/types'
import { toast } from 'sonner'

export function QualityProfilesPage() {
  const { data: profiles, isLoading, isError, refetch } = useQualityProfiles()
  const deleteMutation = useDeleteQualityProfile()

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
          <Button>
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
          action={{ label: 'Add Profile', onClick: () => {} }}
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
                      Cutoff: {getCutoffName(profile.cutoff)}
                    </CardDescription>
                  </div>
                  <div className="flex gap-1">
                    <Button variant="ghost" size="icon">
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
    </div>
  )
}

import { useState } from 'react'

import { Sliders } from 'lucide-react'
import { toast } from 'sonner'

import { IconTile } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { QualityProfileDialog } from '@/components/qualityprofiles'
import { Screen } from '@/components/screen/screen'
import type { SettingsRowAction } from '@/components/settings/settings-item-row'
import { SettingsItemRow } from '@/components/settings/settings-item-row'
import { AddAction, SettingsList } from '@/components/settings/settings-list'
import { useDeleteQualityProfile, useQualityProfiles } from '@/hooks'
import { withToast } from '@/lib/with-toast'
import { getModule } from '@/modules'
import type { QualityProfile } from '@/types'
import { PREDEFINED_QUALITIES } from '@/types'

function useQualityProfilesPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingProfile, setEditingProfile] = useState<QualityProfile | null>(null)
  const query = useQualityProfiles()
  const deleteMutation = useDeleteQualityProfile()

  return {
    query,
    dialogOpen,
    setDialogOpen,
    editingProfile,
    handleAdd: () => {
      setEditingProfile(null)
      setDialogOpen(true)
    },
    handleEdit: (profile: QualityProfile) => {
      setEditingProfile(profile)
      setDialogOpen(true)
    },
    handleDelete: withToast(async (id: number) => {
      await deleteMutation.mutateAsync(id)
      toast.success('Profile deleted')
    }),
  }
}

type PageState = ReturnType<typeof useQualityProfilesPage>

export function QualityProfilesPage() {
  const page = useQualityProfilesPage()
  const back = usePushBack()
  const { data: profiles, isLoading, isError, refetch } = page.query

  return (
    <Screen
      title="Quality Profiles"
      back={back}
      trailing={<AddAction label="Add quality profile" onClick={page.handleAdd} />}
    >
      <SettingsList
        state={{ isLoading, isError, isEmpty: !profiles?.length, refetch }}
        empty="No quality profiles yet"
      >
        {profiles?.map((profile) => (
          <ProfileRow key={profile.id} profile={profile} page={page} />
        ))}
      </SettingsList>
      <QualityProfileDialog
        open={page.dialogOpen}
        onOpenChange={page.setDialogOpen}
        profile={page.editingProfile}
      />
    </Screen>
  )
}

const TILE_BY_MODULE: Record<string, string> = {
  movie: 'bg-movie-600',
  tv: 'bg-tv-600',
}

function ProfileRow({ profile, page }: { profile: QualityProfile; page: PageState }) {
  const actions: SettingsRowAction[] = [
    {
      label: 'Delete',
      destructive: true,
      onClick: () => page.handleDelete(profile.id),
      confirm: {
        title: 'Delete profile',
        description: `Are you sure you want to delete "${profile.name}"?`,
      },
    },
  ]

  return (
    <SettingsItemRow
      leading={
        <IconTile className={TILE_BY_MODULE[profile.moduleType] ?? 'bg-zinc-600'}>
          <Sliders />
        </IconTile>
      }
      title={profile.name}
      subtitle={profileSubtitle(profile)}
      detail={moduleLabel(profile.moduleType)}
      onOpen={() => page.handleEdit(profile)}
      openLabel={`Edit ${profile.name}`}
      actions={actions}
    />
  )
}

function moduleLabel(moduleType: string): string {
  return getModule(moduleType)?.singularName ?? moduleType
}

function profileSubtitle(profile: QualityProfile): string {
  const allowed = profile.items.filter((item) => item.allowed).length
  if (!profile.upgradesEnabled) {
    return `${allowed} qualities · Upgrades disabled`
  }
  const cutoff = PREDEFINED_QUALITIES.find((quality) => quality.id === profile.cutoff)
  return `${allowed} qualities · Cutoff ${cutoff?.name ?? 'Unknown'}`
}

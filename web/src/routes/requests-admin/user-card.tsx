import { Edit, Trash2, Users } from 'lucide-react'

import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import type { PortalUserWithQuota } from '@/types'

import { getProfileName, getQuotaDisplay } from './users-utils'

type UserCardProps = {
  user: PortalUserWithQuota
  qualityProfiles: { id: number; name: string }[] | undefined
  togglePending: boolean
  onToggleEnabled: (user: PortalUserWithQuota) => void
  onEdit: (user: PortalUserWithQuota) => void
  onDelete: (id: number) => void
}

export function UserCard({ user, qualityProfiles, togglePending, onToggleEnabled, onEdit, onDelete }: UserCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between py-4">
        <div className="flex items-center gap-4">
          <div className="bg-muted flex size-10 items-center justify-center rounded-full">
            <Users className="size-5" />
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <CardTitle className="text-base">{user.username}</CardTitle>
              {user.autoApprove ? <Badge>Auto-Approve</Badge> : null}
              {user.enabled ? null : <Badge variant="destructive">Disabled</Badge>}
            </div>
            <CardDescription className="text-xs">
              Movie: {getProfileName(user.movieQualityProfileId, qualityProfiles)} • TV: {getProfileName(user.tvQualityProfileId, qualityProfiles)} •
              Quota: {getQuotaDisplay(user)}
            </CardDescription>
          </div>
        </div>
        <UserCardActions
          user={user}
          togglePending={togglePending}
          onToggleEnabled={onToggleEnabled}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      </CardHeader>
    </Card>
  )
}

type UserCardActionsProps = {
  user: PortalUserWithQuota
  togglePending: boolean
  onToggleEnabled: (user: PortalUserWithQuota) => void
  onEdit: (user: PortalUserWithQuota) => void
  onDelete: (id: number) => void
}

function UserCardActions({ user, togglePending, onToggleEnabled, onEdit, onDelete }: UserCardActionsProps) {
  return (
    <div className="flex items-center gap-4">
      <Switch
        checked={user.enabled}
        onCheckedChange={() => onToggleEnabled(user)}
        disabled={togglePending}
      />
      <Button variant="ghost" size="icon" aria-label="Edit" onClick={() => onEdit(user)}>
        <Edit className="size-4" />
      </Button>
      <ConfirmDialog
        trigger={
          <Button variant="ghost" size="icon" aria-label="Delete">
            <Trash2 className="size-4" />
          </Button>
        }
        title="Delete user"
        description={`Are you sure you want to delete "${user.username}"? Their requests will be preserved.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => onDelete(user.id)}
      />
    </div>
  )
}

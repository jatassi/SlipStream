import { formatDistanceToNow } from 'date-fns'
import { Check, ChevronUp, Copy, Link, RefreshCw, Trash2, UserPlus } from 'lucide-react'

import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { getInvitationLink } from '@/hooks'
import type { Invitation } from '@/types'

import { getInvitationStatus } from './users-utils'

type InvitationCardProps = {
  invitation: Invitation
  copiedToken: string | null
  expandedLinkToken: string | null
  resendPending: boolean
  onCopyLink: (token: string) => void
  onToggleLink: (token: string) => void
  onResend: (id: number) => void
  onDelete: (id: number) => void
}

export function InvitationCard({
  invitation,
  copiedToken,
  expandedLinkToken,
  resendPending,
  onCopyLink,
  onToggleLink,
  onResend,
  onDelete,
}: InvitationCardProps) {
  const isLinkExpanded = expandedLinkToken === invitation.token
  const isUnused = !invitation.usedAt

  return (
    <Card>
      <InvitationCardHeader
        invitation={invitation}
        copiedToken={copiedToken}
        isLinkExpanded={isLinkExpanded}
        isUnused={isUnused}
        resendPending={resendPending}
        onCopyLink={onCopyLink}
        onToggleLink={onToggleLink}
        onResend={onResend}
        onDelete={onDelete}
      />
      {isLinkExpanded && isUnused ? (
        <ExpandedLinkSection token={invitation.token} />
      ) : null}
    </Card>
  )
}

type InvitationCardHeaderProps = {
  invitation: Invitation
  copiedToken: string | null
  isLinkExpanded: boolean
  isUnused: boolean
  resendPending: boolean
  onCopyLink: (token: string) => void
  onToggleLink: (token: string) => void
  onResend: (id: number) => void
  onDelete: (id: number) => void
}

function InvitationCardHeader(props: InvitationCardHeaderProps) {
  const { invitation, isUnused, onDelete } = props

  return (
    <CardHeader className="flex flex-row items-center justify-between py-4">
      <InvitationInfo invitation={invitation} isUnused={isUnused} />
      <div className="flex items-center gap-2">
        {isUnused ? <InvitationActions {...props} /> : null}
        <ConfirmDialog
          trigger={
            <Button variant="ghost" size="icon">
              <Trash2 className="size-4" />
            </Button>
          }
          title="Delete invitation"
          description={`Are you sure you want to delete the invitation for "${invitation.username}"?`}
          confirmLabel="Delete"
          variant="destructive"
          onConfirm={() => onDelete(invitation.id)}
        />
      </div>
    </CardHeader>
  )
}

function InvitationInfo({ invitation, isUnused }: { invitation: Invitation; isUnused: boolean }) {
  const status = getInvitationStatus(invitation)

  return (
    <div className="flex items-center gap-4">
      <div className="bg-muted flex size-10 items-center justify-center rounded-lg">
        <UserPlus className="size-5" />
      </div>
      <div className="space-y-1">
        <div className="flex items-center gap-2">
          <CardTitle className="text-base">{invitation.username}</CardTitle>
          <Badge variant={status.variant}>{status.label}</Badge>
        </div>
        <CardDescription className="text-xs">
          Created{' '}
          {formatDistanceToNow(new Date(invitation.createdAt), { addSuffix: true })}
          {isUnused
            ? ` • Expires ${formatDistanceToNow(new Date(invitation.expiresAt), { addSuffix: true })}`
            : null}
        </CardDescription>
      </div>
    </div>
  )
}

function ExpandedLinkSection({ token }: { token: string }) {
  return (
    <div className="px-6 pb-4">
      <div className="flex items-center gap-2">
        <Input
          readOnly
          value={getInvitationLink(token)}
          className="font-mono text-xs"
          onFocus={(e) => e.target.select()}
        />
      </div>
      <p className="text-muted-foreground mt-2 text-xs">
        Select the link above and copy it manually
      </p>
    </div>
  )
}

type InvitationActionsProps = {
  invitation: Invitation
  copiedToken: string | null
  isLinkExpanded: boolean
  resendPending: boolean
  onCopyLink: (token: string) => void
  onToggleLink: (token: string) => void
  onResend: (id: number) => void
}

function InvitationActions({
  invitation,
  copiedToken,
  isLinkExpanded,
  resendPending,
  onCopyLink,
  onToggleLink,
  onResend,
}: InvitationActionsProps) {
  const CopyIcon = copiedToken === invitation.token ? Check : Copy
  const LinkIcon = isLinkExpanded ? ChevronUp : Link

  return (
    <>
      <Button variant="outline" size="sm" onClick={() => onCopyLink(invitation.token)}>
        <CopyIcon className="mr-1 size-4" />
        Copy Link
      </Button>
      <Button variant="outline" size="sm" onClick={() => onToggleLink(invitation.token)}>
        <LinkIcon className="mr-1 size-4" />
        {isLinkExpanded ? 'Hide' : 'Show'} Link
      </Button>
      <Button
        variant="outline"
        size="sm"
        onClick={() => onResend(invitation.id)}
        disabled={resendPending}
      >
        <RefreshCw className="mr-1 size-4" />
        Resend
      </Button>
    </>
  )
}

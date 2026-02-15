import { Check, Loader2, X } from 'lucide-react'

import { Button } from '@/components/ui/button'

type PlexOAuthButtonProps = {
  hasPlexToken: boolean
  isPlexConnecting: boolean
  actionLabel?: string
  onConnect: () => void
  onDisconnect: () => void
}

export function PlexOAuthButton({
  hasPlexToken,
  isPlexConnecting,
  actionLabel,
  onConnect,
  onDisconnect,
}: PlexOAuthButtonProps) {
  if (hasPlexToken) {
    return (
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
          <Check className="size-4" />
          Connected
        </div>
        <Button variant="ghost" size="sm" onClick={onDisconnect}>
          <X className="size-4" />
        </Button>
      </div>
    )
  }

  if (isPlexConnecting) {
    return (
      <div className="flex items-center gap-2">
        <Button variant="outline" disabled>
          <Loader2 className="mr-2 size-4 animate-spin" />
          Waiting for approval...
        </Button>
      </div>
    )
  }

  return (
    <div className="flex items-center gap-2">
      <Button variant="outline" onClick={onConnect}>
        {actionLabel ?? 'Connect'}
      </Button>
    </div>
  )
}

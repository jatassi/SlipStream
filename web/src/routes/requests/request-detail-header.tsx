import { ArrowLeft, Eye, EyeOff } from 'lucide-react'

import { Button } from '@/components/ui/button'

type RequestDetailHeaderProps = {
  isOwner: boolean
  isWatching: boolean | undefined
  onBack: () => void
  onWatch: () => void
}

export function RequestDetailHeader({
  isOwner,
  isWatching,
  onBack,
  onWatch,
}: RequestDetailHeaderProps) {
  return (
    <div className="flex items-center gap-4">
      <Button variant="ghost" onClick={onBack} className="text-xs md:text-sm">
        <ArrowLeft className="mr-0.5 size-3 md:mr-1 md:size-4" />
        Back
      </Button>
      <div className="flex-1" />
      <WatchButton isOwner={isOwner} isWatching={isWatching} onClick={onWatch} />
    </div>
  )
}

function WatchButton({
  isOwner,
  isWatching,
  onClick,
}: {
  isOwner: boolean
  isWatching: boolean | undefined
  onClick: () => void
}) {
  if (isOwner) {
    return (
      <Button variant="outline" disabled className="text-xs md:text-sm">
        <Eye className="mr-1 size-3 md:mr-2 md:size-4" />
        Watching
      </Button>
    )
  }

  return (
    <Button variant="outline" onClick={onClick} className="text-xs md:text-sm">
      {isWatching ? (
        <EyeOff className="mr-1 size-3 md:mr-2 md:size-4" />
      ) : (
        <Eye className="mr-1 size-3 md:mr-2 md:size-4" />
      )}
      {isWatching ? 'Unwatch' : 'Watch'}
    </Button>
  )
}

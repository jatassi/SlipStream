import { useState } from 'react'

import { Check } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import type { MediaTheme } from './controls-types'
import { themeColor } from './controls-utils'

export function CompletedFlashDemo({ theme }: { theme: MediaTheme }) {
  const [key, setKey] = useState(0)
  const flashClass =
    theme === 'movie'
      ? 'animate-[download-complete-flash-movie_800ms_ease-out]'
      : 'animate-[download-complete-flash-tv_800ms_ease-out]'
  const colorClass = themeColor(theme)

  return (
    <div className="space-y-2">
      <Badge variant="outline">{theme === 'movie' ? 'Movie' : 'TV'}</Badge>
      <div className="flex items-center gap-3">
        <Button
          key={key}
          variant="outline"
          className={cn(flashClass, 'cursor-pointer')}
          onClick={() => setKey((k) => k + 1)}
        >
          <Check className={cn('mr-2 size-4', colorClass)} />
          Downloaded
        </Button>
        <Button
          key={`sm-${key}`}
          variant="outline"
          size="icon-sm"
          className={cn(flashClass, 'cursor-pointer')}
          onClick={() => setKey((k) => k + 1)}
        >
          <Check className={cn('size-4', colorClass)} />
        </Button>
        <span className="text-muted-foreground text-xs">click to replay</span>
      </div>
    </div>
  )
}

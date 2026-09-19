import { cn } from '@/lib/utils'

import type { MediaItem } from './types'

type PosterProps = {
  item: MediaItem
  className?: string
  showTitle?: boolean
  radius?: string
}

function artStyle(item: MediaItem) {
  const [a, b] = item.art
  const seed = item.id % 5
  const orbX = 22 + seed * 14
  const orbY = 18 + ((item.id * 7) % 3) * 12
  return {
    backgroundImage: [
      `radial-gradient(circle at ${orbX}% ${orbY}%, rgba(255,255,255,0.55) 0%, rgba(255,255,255,0.18) 9%, transparent 22%)`,
      `linear-gradient(180deg, transparent 46%, rgba(0,0,0,0.22) 64%, rgba(0,0,0,0.5) 100%)`,
      `radial-gradient(120% 80% at 20% 0%, ${a} 0%, transparent 60%)`,
      `radial-gradient(90% 70% at 90% 100%, ${b} 0%, transparent 65%)`,
      `linear-gradient(160deg, ${a} 0%, ${b} 100%)`,
    ].join(', '),
  }
}

export function Poster({ item, className, showTitle = true, radius = 'rounded-[10px]' }: PosterProps) {
  return (
    <div
      className={cn('relative aspect-[2/3] overflow-hidden [container-type:inline-size]', radius, className)}
      style={artStyle(item)}
      aria-label={item.title}
      role="img"
    >
      <div className="absolute inset-0 bg-[radial-gradient(80%_60%_at_50%_120%,rgba(0,0,0,0.55),transparent)]" />
      <div className="absolute inset-0 shadow-[inset_0_0_0_1px_rgba(255,255,255,0.08)]" />
      {showTitle ? (
        <div className="absolute inset-x-0 bottom-0 p-[9%]">
          <span className="block text-[clamp(10px,11cqw,16px)] leading-[1.1] font-bold tracking-[-0.02em] text-white [text-shadow:0_1px_2px_rgba(0,0,0,0.6)]">
            {item.title}
          </span>
        </div>
      ) : null}
    </div>
  )
}

export function Backdrop({ item, className }: { item: MediaItem; className?: string }) {
  return (
    <div className={cn('relative overflow-hidden', className)} style={artStyle(item)} aria-hidden="true">
      <div className="absolute inset-0 bg-[radial-gradient(60%_50%_at_70%_30%,rgba(255,255,255,0.14),transparent)]" />
      <div className="absolute inset-0 bg-gradient-to-t from-background via-background/40 to-transparent" />
    </div>
  )
}

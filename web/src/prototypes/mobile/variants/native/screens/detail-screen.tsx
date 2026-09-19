import { useState } from 'react'

import { Eye, EyeOff, Star, UserSearch, Zap } from 'lucide-react'

import { cn } from '@/lib/utils'

import { formatGb, KIND_LABEL } from '../../../shared/format'
import { useMobileState } from '../../../shared/mobile-state-context'
import { Backdrop, Poster } from '../../../shared/poster'
import type { MediaItem } from '../../../shared/types'
import { StatusPill } from '../../../shared/ui'
import { Group, Row } from '../grouped-list'
import { NativeScreen } from '../native-screen'

function ActionPill({ icon: Icon, label, active = false, tint, onClick }: { icon: typeof Zap; label: string; active?: boolean; tint: string; onClick?: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'm-press flex h-tap flex-1 items-center justify-center gap-1.5 rounded-full text-m-footnote font-semibold',
        active ? 'text-background' : 'bg-foreground/8 text-foreground',
      )}
      style={active ? { background: tint } : undefined}
    >
      <Icon className="size-4" />
      {label}
    </button>
  )
}

function Hero({ item }: { item: MediaItem }) {
  return (
    <div className="relative">
      <Backdrop item={item} className="h-64" />
      <div className="relative -mt-24 flex items-end gap-4 px-screen">
        <Poster item={item} className="w-28 shrink-0 shadow-[0_10px_30px_rgba(0,0,0,0.5)]" />
        <div className="min-w-0 flex-1 pb-1">
          <div className={cn('text-m-caption font-semibold uppercase tracking-wide', item.kind === 'movie' ? 'text-movie-400' : 'text-tv-400')}>{KIND_LABEL[item.kind]}</div>
          <h1 className="mt-0.5 text-m-heading font-bold text-balance">{item.title}</h1>
          <div className="m-nums mt-1 text-m-footnote text-muted-foreground">{item.year} · {item.length} · {item.genres.join(', ')}</div>
          <div className="mt-2 flex items-center gap-2">
            <StatusPill status={item.status} />
            <span className="m-nums inline-flex items-center gap-1 text-m-caption font-semibold text-amber-400"><Star className="size-3 fill-current" />{item.rating.toFixed(1)}</span>
          </div>
        </div>
      </div>
    </div>
  )
}

function Overview({ text }: { text: string }) {
  const [expanded, setExpanded] = useState(false)
  return (
    <Group header="Overview">
      <button type="button" onClick={() => setExpanded((v) => !v)} className="m-press-row w-full px-4 py-3 text-left">
        <span className={cn('block text-m-body leading-[22px] text-foreground/90', !expanded && 'line-clamp-3')}>{text}</span>
      </button>
    </Group>
  )
}

export function DetailScreen({ id, onBack, backLabel }: { id: number; onBack: () => void; backLabel: string }) {
  const { media, toggleMonitored } = useMobileState()
  const item = media(id)
  const tint = item.kind === 'movie' ? 'var(--movie-500)' : 'var(--tv-500)'

  return (
    <NativeScreen title={item.title} largeTitle={false} back={{ label: backLabel, onClick: onBack }} transparentUntil={200} bottomInset="calc(var(--safe-bottom) + 24px)">
      <Hero item={item} />
      <div className="flex gap-2 px-screen pt-5 pb-7">
        <ActionPill icon={UserSearch} label="Search" tint={tint} />
        <ActionPill icon={Zap} label="Auto" tint={tint} />
        <ActionPill icon={item.monitored ? Eye : EyeOff} label={item.monitored ? 'Monitored' : 'Unmonitored'} active={item.monitored} tint={tint} onClick={() => toggleMonitored(item.id)} />
      </div>
      <Overview text={item.overview} />
      <Group header="File">
        <Row title="Quality" trailing={item.quality} />
        <Row title="Size" trailing={formatGb(item.sizeGb)} />
        <Row title="Profile" trailing={item.profile} />
        <Row title="Added" trailing={item.added} />
      </Group>
      <Group header="Details">
        <Row title="Studio" trailing={item.studio} />
        <Row title="Path" trailing={<span className="max-w-44 truncate font-mono text-[12px]">/mnt/media/{item.kind === 'movie' ? 'movies' : 'tv'}/{item.title} ({item.year})</span>} />
      </Group>
    </NativeScreen>
  )
}

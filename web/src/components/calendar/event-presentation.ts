import { Film, type LucideIcon } from 'lucide-react'

import { getModule } from '@/modules/registry'
import type { CalendarEvent } from '@/types/calendar'

const EVENT_TYPE_LABELS: Record<string, string> = {
  digital: 'Digital release',
  physical: 'Bluray release',
  airDate: 'Air date',
}

const STATUS_LABELS: Record<string, string> = {
  available: 'Available',
  downloading: 'Downloading',
  missing: 'Missing',
}

export function eventKey(event: CalendarEvent): string {
  return `${event.mediaType}-${event.id}-${event.eventType}`
}

function eventModuleId(event: CalendarEvent): string {
  if (event.moduleType) {
    return event.moduleType
  }
  return event.mediaType === 'movie' ? 'movie' : 'tv'
}

export function eventIcon(event: CalendarEvent): LucideIcon {
  return getModule(eventModuleId(event))?.icon ?? Film
}

export function eventTileClass(event: CalendarEvent): string {
  return eventModuleId(event) === 'movie' ? 'bg-movie-600' : 'bg-tv-600'
}

export function eventDotClass(event: CalendarEvent): string {
  return eventModuleId(event) === 'movie' ? 'bg-movie-500' : 'bg-tv-500'
}

export function eventTitle(event: CalendarEvent): string {
  const seriesTitle = event.extra?.seriesTitle as string | undefined
  if (event.mediaType === 'episode' && seriesTitle !== undefined) {
    return seriesTitle
  }
  return event.title
}

function episodeDetail(event: CalendarEvent): string {
  // `||` intentional: Number(undefined) is NaN, which `??` would not catch
  const season = Number(event.extra?.seasonNumber) || 0
  const episode = Number(event.extra?.episodeNumber) || 0
  if (event.title.startsWith('Season ')) {
    return `${event.title} · ${episode} episodes`
  }
  const code = `S${String(season).padStart(2, '0')}E${String(episode).padStart(2, '0')}`
  return `${code} · ${event.title}`
}

export function eventDetail(event: CalendarEvent): string {
  if (event.mediaType === 'episode') {
    return episodeDetail(event)
  }
  return EVENT_TYPE_LABELS[event.eventType] ?? 'Release'
}

export function eventStatusLabel(event: CalendarEvent): string {
  return STATUS_LABELS[event.status] ?? event.status
}

export function eventHref(event: CalendarEvent): string | undefined {
  const mod = getModule(eventModuleId(event))
  if (!mod) {
    return undefined
  }
  const entityId =
    event.mediaType === 'episode' ? (event.extra?.seriesId as number | undefined) : event.id
  if (entityId === undefined) {
    return undefined
  }
  return `${mod.basePath}/${entityId}`
}

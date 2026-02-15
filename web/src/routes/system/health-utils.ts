import type { HealthCategory, HealthItem, HealthStatus } from '@/types/health'

export function getItemNameById(items: HealthItem[], id: string) {
  return items.find((i) => i.id === id)?.name ?? id
}

export function formatRelativeTime(dateString?: string): string {
  if (!dateString) {
    return ''
  }

  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.round(diffMs / 60_000)
  const diffHours = Math.round(diffMs / 3_600_000)
  const diffDays = Math.round(diffMs / 86_400_000)

  if (diffMins < 1) {
    return 'Just now'
  }
  if (diffMins < 60) {
    return `${diffMins} min ago`
  }
  if (diffHours < 24) {
    return `${diffHours} hours ago`
  }
  return `${diffDays} days ago`
}

export function getWorstStatus(items: HealthItem[], initial: HealthStatus = 'ok'): HealthStatus {
  let worst: HealthStatus = initial
  for (const item of items) {
    if (item.status === 'error') {
      return 'error'
    }
    if (item.status === 'warning' && worst !== 'error') {
      worst = 'warning'
    }
  }
  return worst
}

function pluralize(count: number, singular: string, plural: string) {
  return count === 1 ? singular : plural
}

const CATEGORY_VERBS: Record<string, { success: string; failure: string }> = {
  rootFolders: { success: 'accessible', failure: 'inaccessible' },
  metadata: { success: 'responding', failure: 'unreachable' },
}

const DEFAULT_VERBS = { success: 'connected', failure: 'failed' }

const CATEGORY_NOUNS: Record<string, { singular: string; plural: string }> = {
  rootFolders: { singular: 'folder', plural: 'folders' },
  metadata: { singular: 'API', plural: 'APIs' },
}

const DEFAULT_NOUNS = { singular: 'connection', plural: 'connections' }

export type ResultTextParams = {
  category: HealthCategory | 'prowlarr_indexers'
  allItems: HealthItem[]
  resultItems: { id: string }[]
  success: boolean
}

export function getResultText(params: ResultTextParams): string {
  const { category, allItems, resultItems, success } = params
  const count = resultItems.length
  const verbs = CATEGORY_VERBS[category] ?? DEFAULT_VERBS
  const verb = success ? verbs.success : verbs.failure

  if (allItems.length <= 4 && count > 0) {
    const names = resultItems.map((r) => getItemNameById(allItems, r.id)).join(', ')
    return `${names} ${verb}`
  }

  const nouns = CATEGORY_NOUNS[category] ?? DEFAULT_NOUNS
  const noun = pluralize(count, nouns.singular, nouns.plural)

  if (category === 'rootFolders' || category === 'metadata') {
    return `${count} ${noun} ${verb}`
  }
  return `${count} ${noun} ${success ? 'verified' : 'failed'}`
}

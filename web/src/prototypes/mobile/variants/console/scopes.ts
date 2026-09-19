export type Scope = 'board' | 'library' | 'activity' | 'missing' | 'system'

export const SCOPES: { id: Scope; label: string }[] = [
  { id: 'board', label: 'Board' },
  { id: 'library', label: 'Library' },
  { id: 'activity', label: 'Activity' },
  { id: 'missing', label: 'Missing' },
  { id: 'system', label: 'System' },
]

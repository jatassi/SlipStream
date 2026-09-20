export type RequestAction =
  | { kind: 'approve' }
  | { kind: 'approve-manual-search' }
  | { kind: 'approve-auto-search' }
  | { kind: 'deny'; reason: string }
  | { kind: 'delete' }

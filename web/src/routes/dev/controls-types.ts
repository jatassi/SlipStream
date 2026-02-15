export type MediaTheme = 'movie' | 'tv'
export type ControlSize = 'lg' | 'sm' | 'xs'

export type StateName =
  | 'default-monitored'
  | 'default-unmonitored'
  | 'searching-auto'
  | 'searching-manual'
  | 'progress-35'
  | 'progress-72'
  | 'progress-paused'
  | 'completed'
  | 'error-notfound'
  | 'error-failed'

export type DemoState =
  | { type: 'default' }
  | { type: 'searching'; mode: 'manual' | 'auto' }
  | { type: 'progress'; percent: number }
  | { type: 'completed' }
  | { type: 'error'; message: string }

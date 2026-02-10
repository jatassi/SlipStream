export interface LogEntry {
  timestamp: string
  level: string
  component?: string
  message: string
  fields?: Record<string, unknown>
}

export type LogLevel = 'debug' | 'info' | 'warn' | 'error' | 'fatal'

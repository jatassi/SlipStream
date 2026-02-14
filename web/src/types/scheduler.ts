export type ScheduledTask = {
  id: string
  name: string
  description: string
  cron: string
  lastRun?: string
  nextRun?: string
  running: boolean
  lastError?: string
}

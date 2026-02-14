import type { LogEntry } from '@/types/logs'

import { apiFetch, getAdminAuthToken } from './client'

export const logsApi = {
  getRecent: () => apiFetch<LogEntry[]>('/system/logs'),

  downloadLogFile: () => {
    const token = getAdminAuthToken()
    const headers: Record<string, string> = {}
    if (token) {
      headers.Authorization = `Bearer ${token}`
    }
    return fetch('/api/v1/system/logs/download', { headers })
      .then((res) => {
        if (!res.ok) {
          throw new Error('Failed to download log file')
        }
        return res.blob()
      })
      .then((blob) => {
        const url = globalThis.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = 'slipstream.log'
        document.body.append(a)
        a.click()
        globalThis.URL.revokeObjectURL(url)
        a.remove()
      })
  },
}

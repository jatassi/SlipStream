import { apiFetch } from './client'
import type { LogEntry } from '@/types/logs'

export const logsApi = {
  getRecent: () => apiFetch<LogEntry[]>('/system/logs'),

  downloadLogFile: () => {
    const token = localStorage.getItem('admin_token')
    const headers: Record<string, string> = {}
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }
    return fetch('/api/v1/system/logs/download', { headers })
      .then((res) => {
        if (!res.ok) throw new Error('Failed to download log file')
        return res.blob()
      })
      .then((blob) => {
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = 'slipstream.log'
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
      })
  },
}

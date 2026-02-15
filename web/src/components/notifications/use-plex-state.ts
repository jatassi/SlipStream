import { useCallback, useEffect, useRef, useState } from 'react'

import { toast } from 'sonner'

import { apiFetch } from '@/api/client'

import type { PlexSection, PlexServer } from './notification-dialog-types'

type UsePlexStateOptions = {
  isPlex: boolean
  serverId: unknown
  authToken: unknown
  onSettingChange: (name: string, value: unknown) => void
}

export function usePlexState({ isPlex, serverId, authToken, onSettingChange }: UsePlexStateOptions) {
  const [isPlexConnecting, setIsPlexConnecting] = useState(false)
  const [plexServers, setPlexServers] = useState<PlexServer[]>([])
  const [plexSections, setPlexSections] = useState<PlexSection[]>([])
  const [isLoadingServers, setIsLoadingServers] = useState(false)
  const [isLoadingSections, setIsLoadingSections] = useState(false)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const cleanupPolling = useCallback(() => {
    if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null }
    setIsPlexConnecting(false)
  }, [])

  const fetchServers = useCallback(async (token: string) => {
    setIsLoadingServers(true)
    try { setPlexServers(await apiFetch<PlexServer[]>('/notifications/plex/servers', { headers: { 'X-Plex-Token': token } })) }
    catch { /* non-critical */ }
    finally { setIsLoadingServers(false) }
  }, [])

  const fetchSections = useCallback(async (sid: string, token: string) => {
    setIsLoadingSections(true)
    try { setPlexSections(await apiFetch<PlexSection[]>(`/notifications/plex/servers/${sid}/sections`, { headers: { 'X-Plex-Token': token } })) }
    catch { /* non-critical */ }
    finally { setIsLoadingSections(false) }
  }, [])

  useEffect(() => {
    if (isPlex && (serverId as string) && (authToken as string)) { void fetchSections(serverId as string, authToken as string) }
  }, [isPlex, serverId, authToken, fetchSections])

  const startOAuth = useCallback(async () => {
    setIsPlexConnecting(true)
    try { await runPlexOAuthFlow({ pollRef, cleanupPolling, onSettingChange, fetchServers }) }
    catch { setIsPlexConnecting(false); toast.error('Failed to start Plex authentication') }
  }, [cleanupPolling, onSettingChange, fetchServers])

  const disconnect = useCallback(() => {
    for (const key of ['authToken', 'clientId', 'serverId'] as const) { onSettingChange(key, '') }
    onSettingChange('sectionIds', [])
    setPlexServers([]); setPlexSections([])
  }, [onSettingChange])

  const resetState = useCallback(() => { setPlexServers([]); setPlexSections([]) }, [])

  return {
    isPlexConnecting, plexServers, plexSections, isLoadingServers, isLoadingSections,
    cleanupPolling, fetchServers, startOAuth, disconnect, resetState,
  }
}

type OAuthFlowOptions = {
  pollRef: React.RefObject<ReturnType<typeof setInterval> | null>
  cleanupPolling: () => void
  onSettingChange: (name: string, value: unknown) => void
  fetchServers: (token: string) => Promise<void>
}

async function runPlexOAuthFlow({ pollRef, cleanupPolling, onSettingChange, fetchServers }: OAuthFlowOptions) {
  const { pinId, authUrl, clientId } = await apiFetch<{ pinId: number; authUrl: string; clientId: string }>(
    '/notifications/plex/auth/start', { method: 'POST' },
  )
  onSettingChange('clientId', clientId)
  window.open(authUrl, '_blank', 'width=800,height=600')

  ;(pollRef as { current: ReturnType<typeof setInterval> | null }).current = setInterval(async () => {
    try {
      const status = await apiFetch<{ complete: boolean; authToken?: string }>(`/notifications/plex/auth/status/${pinId}`)
      if (status.complete && status.authToken) {
        cleanupPolling()
        onSettingChange('authToken', status.authToken)
        toast.success('Connected to Plex!')
        void fetchServers(status.authToken)
      }
    } catch (error) {
      if (error && typeof error === 'object' && 'status' in error && error.status === 410) {
        cleanupPolling()
        toast.error('Plex authentication expired. Please try again.')
      }
    }
  }, 2000)

  setTimeout(() => {
    if (pollRef.current) { cleanupPolling(); toast.error('Plex authentication timed out. Please try again.') }
  }, 5 * 60 * 1000)
}

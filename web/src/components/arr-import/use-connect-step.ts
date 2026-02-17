import { useState } from 'react'

import { useConnect, useDetectDB } from '@/hooks/use-arr-import'
import type { SourceType } from '@/types/arr-import'

export type ConnectionMethod = 'sqlite' | 'api'

export function useConnectStep(sourceType: SourceType, onConnected: () => void) {
  const [connectionMethod, setConnectionMethod] = useState<ConnectionMethod>('sqlite')
  const [dbPath, setDbPath] = useState('')
  const [url, setUrl] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [browserOpen, setBrowserOpen] = useState(false)

  const detectQuery = useDetectDB(sourceType, connectionMethod === 'sqlite')

  const [prevFound, setPrevFound] = useState<string | undefined>(undefined)
  if (detectQuery.data?.found !== prevFound) {
    setPrevFound(detectQuery.data?.found)
    if (detectQuery.data?.found && !dbPath) {
      setDbPath(detectQuery.data.found)
    }
  }

  const connectMutation = useConnect()

  const handleConnect = () => {
    const config = {
      sourceType,
      ...(connectionMethod === 'sqlite' ? { dbPath } : { url, apiKey }),
    }
    connectMutation.mutate(config, {
      onSuccess: () => {
        onConnected()
      },
    })
  }

  const isValid =
    connectionMethod === 'sqlite' ? dbPath.trim() !== '' : url.trim() !== '' && apiKey.trim() !== ''

  return {
    connectionMethod,
    setConnectionMethod,
    dbPath,
    setDbPath,
    url,
    setUrl,
    apiKey,
    setApiKey,
    browserOpen,
    setBrowserOpen,
    connectMutation,
    handleConnect,
    isValid,
    detectQuery,
  }
}

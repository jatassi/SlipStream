import { useState } from 'react'

import { useNavigate } from '@tanstack/react-router'

import { usePasskeyCredentials, usePasskeySupport } from '@/hooks/portal'
import { usePortalAuthStore } from '@/stores'

import { PasskeyPromptModal } from './passkey-prompt-modal'

const DISMISSED_STORAGE_KEY = 'slipstream_passkey_prompt_dismissed'

function isPermanentlyDismissed(): boolean {
  try {
    return localStorage.getItem(DISMISSED_STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

function persistDismissal(dontShowAgain: boolean) {
  if (!dontShowAgain) {
    return
  }
  try {
    localStorage.setItem(DISMISSED_STORAGE_KEY, '1')
  } catch {
    /* storage unavailable */
  }
}

function shouldShowPrompt(args: {
  isSupported: boolean
  credentialCount: number
}): boolean {
  return args.isSupported && !isPermanentlyDismissed() && args.credentialCount === 0
}

function usePasskeyPromptTrigger() {
  const { isAuthenticated, justLoggedIn, consumeJustLoggedIn } = usePortalAuthStore()
  const { isSupported } = usePasskeySupport()
  const { data: credentials, isLoading: credentialsLoading } = usePasskeyCredentials()
  const [open, setOpen] = useState(false)
  const [evaluated, setEvaluated] = useState(false)

  // Render-time state adjustment: evaluate exactly once per sign-in event,
  // gated on credential data being loaded.
  const ready = isAuthenticated && justLoggedIn && !credentialsLoading
  if (ready && !evaluated) {
    setEvaluated(true)
    consumeJustLoggedIn()
    if (shouldShowPrompt({ isSupported, credentialCount: credentials?.length ?? 0 })) {
      setOpen(true)
    }
  }
  if (!isAuthenticated && evaluated) {
    setEvaluated(false)
  }

  return { open, setOpen }
}

export function PasskeyPromptHost() {
  const navigate = useNavigate()
  const { open, setOpen } = usePasskeyPromptTrigger()

  const handleDismiss = (dontShowAgain: boolean) => {
    persistDismissal(dontShowAgain)
    setOpen(false)
  }

  const handleCreate = (dontShowAgain: boolean) => {
    persistDismissal(dontShowAgain)
    setOpen(false)
    void navigate({ to: '/requests/settings', hash: 'new-passkey' })
  }

  return (
    <PasskeyPromptModal
      open={open}
      onOpenChange={setOpen}
      onDismiss={handleDismiss}
      onCreate={handleCreate}
    />
  )
}

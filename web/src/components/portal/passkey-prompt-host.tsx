import { useEffect, useState } from 'react'

import { useNavigate } from '@tanstack/react-router'

import { usePasskeyCredentials, usePasskeySupport } from '@/hooks/portal'
import { usePortalAuthStore } from '@/stores'

import { NEW_PASSKEY_HASH } from './passkey-deep-link'
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
  const { isAuthenticated, justLoggedIn, user, consumeJustLoggedIn } = usePortalAuthStore()
  const { isSupported } = usePasskeySupport()
  const { data: credentials, isLoading: credentialsLoading } = usePasskeyCredentials()
  const [open, setOpen] = useState(false)
  const [evaluated, setEvaluated] = useState(false)

  const ready = isAuthenticated && justLoggedIn && !user?.isAdmin && !credentialsLoading
  if (ready && !evaluated) {
    setEvaluated(true)
    if (shouldShowPrompt({ isSupported, credentialCount: credentials?.length ?? 0 })) {
      setOpen(true)
    }
  }

  useEffect(() => {
    if (evaluated && justLoggedIn) {
      consumeJustLoggedIn()
    }
  }, [evaluated, justLoggedIn, consumeJustLoggedIn])

  return { open, setOpen }
}

export function PasskeyPromptHost() {
  const navigate = useNavigate()
  const { open, setOpen } = usePasskeyPromptTrigger()

  const close = (dontShowAgain: boolean) => {
    persistDismissal(dontShowAgain)
    setOpen(false)
  }

  return (
    <PasskeyPromptModal
      open={open}
      onOpenChange={setOpen}
      onDismiss={close}
      onCreate={(dontShowAgain) => {
        close(dontShowAgain)
        void navigate({ to: '/requests/settings', hash: NEW_PASSKEY_HASH })
      }}
    />
  )
}

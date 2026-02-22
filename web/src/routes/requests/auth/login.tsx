import { useCallback, useEffect, useRef, useState } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { Ban, KeyRound, Loader2, Trash2, User } from 'lucide-react'
import { toast } from 'sonner'

import { deleteAdmin } from '@/api/auth'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { InputOTP, InputOTPGroup, InputOTPSlot } from '@/components/ui/input-otp'
import { Label } from '@/components/ui/label'
import { LoadingButton } from '@/components/ui/loading-button'
import { useAuthStatus, usePortalEnabled, usePortalLogin } from '@/hooks'
import { usePasskeyLogin, usePasskeySupport } from '@/hooks/portal'
import { usePortalAuthStore } from '@/stores'

const GRADIENT_BORDER = { borderImage: 'linear-gradient(to right, var(--movie-500), var(--tv-500)) 1' }

function AuthPageShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-background flex min-h-screen items-center justify-center p-4">
      {children}
    </div>
  )
}

function PortalDisabledView() {
  return (
    <AuthPageShell>
      <Card className="w-full max-w-md">
        <CardContent className="pt-6">
          <div className="space-y-4 text-center">
            <Ban className="text-muted-foreground mx-auto size-12" />
            <h1 className="text-xl font-semibold">Requests Portal Disabled</h1>
            <p className="text-muted-foreground">
              The external requests portal is currently disabled. Please contact your server administrator.
            </p>
          </div>
        </CardContent>
      </Card>
    </AuthPageShell>
  )
}

type PasskeySectionProps = {
  onLogin: () => void
  isPending: boolean
  onUsePinInstead: () => void
}

function PasskeySection({ onLogin, isPending, onUsePinInstead }: PasskeySectionProps) {
  return (
    <div className="space-y-6">
      <LoadingButton loading={isPending} icon={KeyRound} iconClassName="mr-1 size-3 md:mr-2 md:size-4" onClick={onLogin} className="w-full text-sm md:text-base">
        Sign in with Passkey
      </LoadingButton>
      <div className="text-center">
        <button type="button" onClick={onUsePinInstead} className="text-muted-foreground hover:text-foreground text-sm hover:underline">
          Use PIN instead
        </button>
      </div>
    </div>
  )
}

type UsernameFieldProps = {
  showInput: boolean
  username: string
  onUsernameChange: (value: string) => void
  onSwitchUser: () => void
  inputRef: React.RefObject<HTMLInputElement | null>
}

function UsernameField({ showInput, username, onUsernameChange, onSwitchUser, inputRef }: UsernameFieldProps) {
  if (showInput) {
    return (
      <div className="space-y-2">
        <Label htmlFor="username">Username</Label>
        <Input ref={inputRef} id="username" type="text" placeholder="Your username" value={username} onChange={(e) => onUsernameChange(e.target.value)} required autoComplete="username" />
      </div>
    )
  }
  return (
    <div className="border-border bg-muted/50 flex items-center justify-between rounded-lg border p-2 md:p-3">
      <div className="flex items-center gap-2 md:gap-3">
        <div className="bg-primary/10 rounded-full p-1.5 md:p-2">
          <User className="text-primary size-4 md:size-5" />
        </div>
        <span className="text-sm font-medium md:text-base">{username}</span>
      </div>
      <Button type="button" variant="ghost" size="sm" onClick={onSwitchUser} className="text-xs md:text-sm">
        Switch User
      </Button>
    </div>
  )
}

type PinFormProps = {
  username: string
  showUsernameInput: boolean
  onUsernameChange: (value: string) => void
  onSwitchUser: () => void
  usernameInputRef: React.RefObject<HTMLInputElement | null>
  pin: string
  onPinChange: (value: string) => void
  onSubmit: (e: React.SyntheticEvent) => void
  isPending: boolean
  passkeySupported: boolean
  onUsePasskey: () => void
}

function PinLoginForm(props: PinFormProps) {
  return (
    <>
      <form onSubmit={props.onSubmit} className="space-y-6">
        <UsernameField showInput={props.showUsernameInput} username={props.username} onUsernameChange={props.onUsernameChange} onSwitchUser={props.onSwitchUser} inputRef={props.usernameInputRef} />
        <div className="space-y-3">
          <Label>PIN</Label>
          <div className="flex justify-center">
            <InputOTP mask maxLength={4} value={props.pin} onChange={props.onPinChange}>
              <InputOTPGroup className="gap-2 *:data-[slot=input-otp-slot]:rounded-md *:data-[slot=input-otp-slot]:border md:gap-2.5">
                <InputOTPSlot index={0} className="size-10 text-lg md:size-12 md:text-xl" />
                <InputOTPSlot index={1} className="size-10 text-lg md:size-12 md:text-xl" />
                <InputOTPSlot index={2} className="size-10 text-lg md:size-12 md:text-xl" />
                <InputOTPSlot index={3} className="size-10 text-lg md:size-12 md:text-xl" />
              </InputOTPGroup>
            </InputOTP>
          </div>
        </div>
        <LoadingButton type="submit" className="w-full text-sm md:text-base" loading={props.isPending} disabled={props.pin.length !== 4 || props.username.trim() === ''}>
          Sign In
        </LoadingButton>
      </form>
      {props.passkeySupported ? (
        <div className="mt-4 text-center">
          <button type="button" onClick={props.onUsePasskey} className="text-muted-foreground hover:text-foreground text-sm hover:underline">
            Use Passkey instead
          </button>
        </div>
      ) : null}
    </>
  )
}

type DebugDeleteProps = {
  showDelete: boolean
  isDeleting: boolean
  onDelete: () => void
}

function DebugDeleteSection({ showDelete, isDeleting, onDelete }: DebugDeleteProps) {
  if (!showDelete) {return null}
  return (
    <div className="border-border mt-6 border-t pt-4">
      <LoadingButton type="button" loading={isDeleting} icon={Trash2} iconClassName="mr-1 size-3 md:mr-2 md:size-4" variant="destructive" size="sm" className="w-full text-xs md:text-sm" onClick={onDelete}>
        Delete Admin (Debug)
      </LoadingButton>
    </div>
  )
}

function useLoginPage() {
  const navigate = useNavigate()
  const { getPostLoginRedirect } = usePortalAuthStore()
  const loginMutation = usePortalLogin()
  const passkeyLoginMutation = usePasskeyLogin()
  const { data: authStatus, refetch: refetchAuthStatus } = useAuthStatus()
  const { isSupported: passkeySupported, isLoading: passkeyLoading } = usePasskeySupport()
  const portalEnabled = usePortalEnabled()

  const rememberedUsername = localStorage.getItem('slipstream_last_username') ?? ''
  const [username, setUsername] = useState(rememberedUsername)
  const [showUsernameInput, setShowUsernameInput] = useState(!rememberedUsername)
  const [showPinForm, setShowPinForm] = useState(false)
  const [pin, setPin] = useState('')
  const [isDeleting, setIsDeleting] = useState(false)
  const usernameInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => { if (showUsernameInput) {usernameInputRef.current?.focus()} }, [showUsernameInput])

  const performLogin = useCallback(() => {
    if (loginMutation.isPending) {return}
    loginMutation.mutate(
      { username, password: pin },
      {
        onSuccess: () => { localStorage.setItem('slipstream_last_username', username); void navigate({ to: getPostLoginRedirect() }) },
        onError: (error) => { toast.error('Login failed', { description: error.message || 'Invalid credentials' }); setPin('') },
      },
    )
  }, [username, pin, loginMutation, getPostLoginRedirect, navigate])

  useEffect(() => { if (pin.length === 4 && username.trim() !== '') {performLogin()} }, [pin, username, performLogin])

  return {
    portalEnabled, passkeyLoading, passkeySupported,
    shouldShowPasskeyLogin: !passkeyLoading && passkeySupported && !showPinForm,
    username, setUsername, showUsernameInput, usernameInputRef,
    pin, setPin, isDeleting, loginPending: loginMutation.isPending,
    showDelete: !authStatus?.requiresSetup,
    handleSwitchUser: () => { setUsername(''); setPin(''); setShowUsernameInput(true) },
    handleSubmit: (e: React.SyntheticEvent) => { e.preventDefault(); performLogin() },
    handlePasskeyLogin: () => {
      passkeyLoginMutation.mutate(undefined, { onSuccess: () => void navigate({ to: getPostLoginRedirect() }) })
    },
    passkeyLoginPending: passkeyLoginMutation.isPending,
    setShowPinForm,
    handleDeleteAdmin: async () => {
      setIsDeleting(true)
      try { await deleteAdmin(); toast.success('Admin deleted'); await refetchAuthStatus(); void navigate({ to: '/auth/setup' }) }
      catch { toast.error('Failed to delete admin') }
      finally { setIsDeleting(false) }
    },
  }
}

export function LoginPage() {
  const vm = useLoginPage()

  if (!vm.portalEnabled) {return <PortalDisabledView />}

  return (
    <AuthPageShell>
      <Card className="w-full max-w-md border-t-2 border-t-transparent" style={GRADIENT_BORDER}>
        <CardHeader className="text-center">
          <CardTitle className="text-media-gradient text-2xl">Welcome Back</CardTitle>
          <CardDescription>Sign in to your SlipStream account</CardDescription>
        </CardHeader>
        <CardContent>
          {vm.passkeyLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="text-muted-foreground size-6 animate-spin" />
            </div>
          ) : null}
          {!vm.passkeyLoading && vm.shouldShowPasskeyLogin ? (
            <PasskeySection onLogin={vm.handlePasskeyLogin} isPending={vm.passkeyLoginPending} onUsePinInstead={() => vm.setShowPinForm(true)} />
          ) : null}
          {!vm.passkeyLoading && !vm.shouldShowPasskeyLogin ? (
            <PinLoginForm
              username={vm.username} showUsernameInput={vm.showUsernameInput} onUsernameChange={vm.setUsername}
              onSwitchUser={vm.handleSwitchUser} usernameInputRef={vm.usernameInputRef}
              pin={vm.pin} onPinChange={vm.setPin} onSubmit={vm.handleSubmit}
              isPending={vm.loginPending} passkeySupported={vm.passkeySupported} onUsePasskey={() => vm.setShowPinForm(false)}
            />
          ) : null}
          <DebugDeleteSection showDelete={vm.showDelete} isDeleting={vm.isDeleting} onDelete={vm.handleDeleteAdmin} />
        </CardContent>
      </Card>
    </AuthPageShell>
  )
}

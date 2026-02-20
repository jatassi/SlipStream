import { useCallback, useEffect, useState } from 'react'

import { useNavigate, useSearch } from '@tanstack/react-router'
import { AlertCircle, Ban, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { InputOTP, InputOTPGroup, InputOTPSlot } from '@/components/ui/input-otp'
import { Label } from '@/components/ui/label'
import { usePortalEnabled, usePortalSignup, useValidateInvitation } from '@/hooks'

const GRADIENT_BORDER = { borderImage: 'linear-gradient(to right, var(--movie-500), var(--tv-500)) 1' }

function AuthShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-background flex min-h-screen items-center justify-center p-4">
      {children}
    </div>
  )
}

function PortalDisabledView() {
  return (
    <AuthShell>
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
    </AuthShell>
  )
}

function NoTokenView() {
  return (
    <AuthShell>
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Sign Up</CardTitle>
          <CardDescription>Create your SlipStream account</CardDescription>
        </CardHeader>
        <CardContent>
          <Alert variant="destructive">
            <AlertCircle className="size-4" />
            <AlertDescription>No invitation token provided. You need an invitation link to create an account.</AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </AuthShell>
  )
}

function ValidatingView() {
  return (
    <div className="bg-background flex min-h-screen items-center justify-center">
      <div className="flex flex-col items-center gap-2">
        <Loader2 className="text-muted-foreground size-6 animate-spin md:size-8" />
        <p className="text-muted-foreground text-sm md:text-base">Validating invitation...</p>
      </div>
    </div>
  )
}

function InvalidInvitationView() {
  return (
    <AuthShell>
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Invalid Invitation</CardTitle>
        </CardHeader>
        <CardContent>
          <Alert variant="destructive">
            <AlertCircle className="size-4" />
            <AlertDescription>This invitation link is invalid or has expired. Please request a new invitation.</AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </AuthShell>
  )
}

function PinOTPInput({ pin, onPinChange }: { pin: string; onPinChange: (value: string) => void }) {
  return (
    <div className="flex justify-center">
      <InputOTP mask maxLength={4} value={pin} onChange={onPinChange}>
        <InputOTPGroup className="gap-2 *:data-[slot=input-otp-slot]:rounded-md *:data-[slot=input-otp-slot]:border md:gap-2.5">
          <InputOTPSlot index={0} className="size-10 text-lg md:size-12 md:text-xl" />
          <InputOTPSlot index={1} className="size-10 text-lg md:size-12 md:text-xl" />
          <InputOTPSlot index={2} className="size-10 text-lg md:size-12 md:text-xl" />
          <InputOTPSlot index={3} className="size-10 text-lg md:size-12 md:text-xl" />
        </InputOTPGroup>
      </InputOTP>
    </div>
  )
}

function useSignup(token: string) {
  const navigate = useNavigate()
  const signupMutation = usePortalSignup()
  const [pin, setPin] = useState('')

  const signupMutate = signupMutation.mutate
  const signupPending = signupMutation.isPending

  const performSignup = useCallback((username: string) => {
    if (signupPending) {return}
    signupMutate(
      { token, password: pin },
      {
        onSuccess: () => {
          localStorage.setItem('slipstream_last_username', username)
          toast.success('Account created successfully')
          void navigate({ to: '/requests' })
        },
        onError: (error) => { toast.error('Signup failed', { description: error.message }); setPin('') },
      },
    )
  }, [token, pin, signupMutate, signupPending, navigate])

  return { pin, setPin, performSignup, isPending: signupPending }
}

type SignupFormProps = {
  username: string
  signup: ReturnType<typeof useSignup>
}

function SignupForm({ username, signup }: SignupFormProps) {
  return (
    <AuthShell>
      <Card className="w-full max-w-md border-t-2 border-t-transparent" style={GRADIENT_BORDER}>
        <CardHeader className="text-center">
          <CardTitle className="text-media-gradient text-2xl">Welcome, {username}!</CardTitle>
          <CardDescription>Create a 4-digit PIN to secure your account</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={(e) => { e.preventDefault(); signup.performSignup(username) }} className="space-y-6">
            <div className="space-y-3">
              <Label>PIN</Label>
              <PinOTPInput pin={signup.pin} onPinChange={signup.setPin} />
              <p className="text-muted-foreground text-center text-xs">Choose a 4-digit PIN you&apos;ll remember</p>
            </div>
            <Button type="submit" className="w-full text-sm md:text-base" disabled={signup.isPending || signup.pin.length !== 4}>
              {signup.isPending ? <Loader2 className="mr-1 size-3 animate-spin md:mr-2 md:size-4" /> : null}
              Create Account
            </Button>
          </form>
        </CardContent>
      </Card>
    </AuthShell>
  )
}

export function SignupPage() {
  const searchParams: { token?: string } = useSearch({ strict: false })
  const token = searchParams.token ?? ''

  const { data: invitation, isLoading: validating, error: validationError } = useValidateInvitation(token)
  const portalEnabled = usePortalEnabled()
  const signup = useSignup(token)

  useEffect(() => { if (signup.pin.length === 4 && invitation?.username) { signup.performSignup(invitation.username) } }, [signup, invitation?.username])

  if (!portalEnabled) { return <PortalDisabledView /> }
  if (!token) { return <NoTokenView /> }
  if (validating) { return <ValidatingView /> }
  if (validationError || !invitation?.valid) { return <InvalidInvitationView /> }

  return <SignupForm username={invitation.username} signup={signup} />
}

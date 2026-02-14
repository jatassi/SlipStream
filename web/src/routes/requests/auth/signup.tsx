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

type SignupSearchParams = {
  token?: string
}

export function SignupPage() {
  const navigate = useNavigate()
  const search = useSearch({ from: '/requests/auth/signup' })
  const token = search.token || ''

  const {
    data: invitation,
    isLoading: validating,
    error: validationError,
  } = useValidateInvitation(token)
  const signupMutation = usePortalSignup()
  const portalEnabled = usePortalEnabled()

  const [pin, setPin] = useState('')

  const signupMutate = signupMutation.mutate
  const signupPending = signupMutation.isPending
  const invitationUsername = invitation?.username

  const performSignup = useCallback(() => {
    if (signupPending) {
      return
    }

    signupMutate(
      { token, password: pin },
      {
        onSuccess: () => {
          localStorage.setItem('slipstream_last_username', invitationUsername || '')
          toast.success('Account created successfully')
          navigate({ to: '/requests' })
        },
        onError: (error) => {
          toast.error('Signup failed', {
            description: error.message || 'Could not create account',
          })
          setPin('')
        },
      },
    )
  }, [token, pin, signupMutate, signupPending, invitationUsername, navigate])

  useEffect(() => {
    if (pin.length === 4) {
      performSignup()
    }
  }, [pin, performSignup])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    performSignup()
  }

  if (!portalEnabled) {
    return (
      <div className="bg-background flex min-h-screen items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardContent className="pt-6">
            <div className="space-y-4 text-center">
              <Ban className="text-muted-foreground mx-auto size-12" />
              <h1 className="text-xl font-semibold">Requests Portal Disabled</h1>
              <p className="text-muted-foreground">
                The external requests portal is currently disabled. Please contact your server
                administrator.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!token) {
    return (
      <div className="bg-background flex min-h-screen items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <CardTitle className="text-2xl">Sign Up</CardTitle>
            <CardDescription>Create your SlipStream account</CardDescription>
          </CardHeader>
          <CardContent>
            <Alert variant="destructive">
              <AlertCircle className="size-4" />
              <AlertDescription>
                No invitation token provided. You need an invitation link to create an account.
              </AlertDescription>
            </Alert>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (validating) {
    return (
      <div className="bg-background flex min-h-screen items-center justify-center">
        <div className="flex flex-col items-center gap-2">
          <Loader2 className="text-muted-foreground size-6 animate-spin md:size-8" />
          <p className="text-muted-foreground text-sm md:text-base">Validating invitation...</p>
        </div>
      </div>
    )
  }

  if (validationError || !invitation?.valid) {
    return (
      <div className="bg-background flex min-h-screen items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <CardTitle className="text-2xl">Invalid Invitation</CardTitle>
          </CardHeader>
          <CardContent>
            <Alert variant="destructive">
              <AlertCircle className="size-4" />
              <AlertDescription>
                This invitation link is invalid or has expired. Please request a new invitation.
              </AlertDescription>
            </Alert>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="bg-background flex min-h-screen items-center justify-center p-4">
      <Card
        className="w-full max-w-md border-t-2 border-t-transparent"
        style={{ borderImage: 'linear-gradient(to right, var(--movie-500), var(--tv-500)) 1' }}
      >
        <CardHeader className="text-center">
          <CardTitle className="text-media-gradient text-2xl">
            Welcome, {invitation.username}!
          </CardTitle>
          <CardDescription>Create a 4-digit PIN to secure your account</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="space-y-3">
              <Label>PIN</Label>
              <div className="flex justify-center">
                <InputOTP maxLength={4} value={pin} onChange={setPin} autoFocus>
                  <InputOTPGroup className="gap-2 *:data-[slot=input-otp-slot]:rounded-md *:data-[slot=input-otp-slot]:border md:gap-2.5">
                    <InputOTPSlot index={0} className="size-10 text-lg md:size-12 md:text-xl" />
                    <InputOTPSlot index={1} className="size-10 text-lg md:size-12 md:text-xl" />
                    <InputOTPSlot index={2} className="size-10 text-lg md:size-12 md:text-xl" />
                    <InputOTPSlot index={3} className="size-10 text-lg md:size-12 md:text-xl" />
                  </InputOTPGroup>
                </InputOTP>
              </div>
              <p className="text-muted-foreground text-center text-xs">
                Choose a 4-digit PIN you'll remember
              </p>
            </div>
            <Button
              type="submit"
              className="w-full text-sm md:text-base"
              disabled={signupMutation.isPending || pin.length !== 4}
            >
              {signupMutation.isPending ? (
                <Loader2 className="mr-1 size-3 animate-spin md:mr-2 md:size-4" />
              ) : null}
              Create Account
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

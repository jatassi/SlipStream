import { useCallback, useEffect, useState } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { Ban, KeyRound, Loader2, Trash2, User } from 'lucide-react'
import { toast } from 'sonner'

import { deleteAdmin } from '@/api/auth'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { InputOTP, InputOTPGroup, InputOTPSlot } from '@/components/ui/input-otp'
import { Label } from '@/components/ui/label'
import { useAuthStatus, usePortalEnabled, usePortalLogin } from '@/hooks'
import { usePasskeyLogin, usePasskeySupport } from '@/hooks/portal'
import { usePortalAuthStore } from '@/stores'

export function LoginPage() {
  const navigate = useNavigate()
  const { getPostLoginRedirect } = usePortalAuthStore()
  const loginMutation = usePortalLogin()
  const passkeyLoginMutation = usePasskeyLogin()
  const { data: authStatus, refetch: refetchAuthStatus } = useAuthStatus()
  const { isSupported: passkeySupported, isLoading: passkeyLoading } = usePasskeySupport()
  const portalEnabled = usePortalEnabled()

  const rememberedUsername = localStorage.getItem('slipstream_last_username') || ''
  const [username, setUsername] = useState(rememberedUsername)
  const [showUsernameInput, setShowUsernameInput] = useState(!rememberedUsername)
  const [showPinForm, setShowPinForm] = useState(false)
  const [pin, setPin] = useState('')
  const [isDeleting, setIsDeleting] = useState(false)

  const shouldShowPasskeyLogin = !passkeyLoading && passkeySupported && !showPinForm

  const handleSwitchUser = () => {
    setUsername('')
    setPin('')
    setShowUsernameInput(true)
  }

  const handleDeleteAdmin = async () => {
    setIsDeleting(true)
    try {
      await deleteAdmin()
      toast.success('Admin deleted')
      await refetchAuthStatus()
      navigate({ to: '/auth/setup' })
    } catch {
      toast.error('Failed to delete admin')
    } finally {
      setIsDeleting(false)
    }
  }

  const performLogin = useCallback(() => {
    if (loginMutation.isPending) {
      return
    }

    loginMutation.mutate(
      { username, password: pin },
      {
        onSuccess: () => {
          localStorage.setItem('slipstream_last_username', username)
          const redirect = getPostLoginRedirect()
          navigate({ to: redirect })
        },
        onError: (error) => {
          toast.error('Login failed', {
            description: error.message || 'Invalid credentials',
          })
          setPin('')
        },
      },
    )
  }, [username, pin, loginMutation, getPostLoginRedirect, navigate])

  useEffect(() => {
    if (pin.length === 4 && username.trim() !== '') {
      performLogin()
    }
  }, [pin, username, performLogin])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    performLogin()
  }

  const handlePasskeyLogin = () => {
    passkeyLoginMutation.mutate(undefined, {
      onSuccess: () => {
        const redirect = getPostLoginRedirect()
        navigate({ to: redirect })
      },
    })
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

  return (
    <div className="bg-background flex min-h-screen items-center justify-center p-4">
      <Card
        className="w-full max-w-md border-t-2 border-t-transparent"
        style={{ borderImage: 'linear-gradient(to right, var(--movie-500), var(--tv-500)) 1' }}
      >
        <CardHeader className="text-center">
          <CardTitle className="text-media-gradient text-2xl">Welcome Back</CardTitle>
          <CardDescription>Sign in to your SlipStream account</CardDescription>
        </CardHeader>
        <CardContent>
          {passkeyLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="text-muted-foreground size-6 animate-spin" />
            </div>
          ) : shouldShowPasskeyLogin ? (
            <div className="space-y-6">
              <Button
                onClick={handlePasskeyLogin}
                disabled={passkeyLoginMutation.isPending}
                className="w-full text-sm md:text-base"
              >
                {passkeyLoginMutation.isPending ? (
                  <Loader2 className="mr-1 size-3 animate-spin md:mr-2 md:size-4" />
                ) : (
                  <KeyRound className="mr-1 size-3 md:mr-2 md:size-4" />
                )}
                Sign in with Passkey
              </Button>

              <div className="text-center">
                <button
                  type="button"
                  onClick={() => setShowPinForm(true)}
                  className="text-muted-foreground hover:text-foreground text-sm hover:underline"
                >
                  Use PIN instead
                </button>
              </div>
            </div>
          ) : (
            <>
              <form onSubmit={handleSubmit} className="space-y-6">
                {showUsernameInput ? (
                  <div className="space-y-2">
                    <Label htmlFor="username">Username</Label>
                    <Input
                      id="username"
                      type="text"
                      placeholder="Your username"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      required
                      autoComplete="username"
                      autoFocus
                    />
                  </div>
                ) : (
                  <div className="border-border bg-muted/50 flex items-center justify-between rounded-lg border p-2 md:p-3">
                    <div className="flex items-center gap-2 md:gap-3">
                      <div className="bg-primary/10 rounded-full p-1.5 md:p-2">
                        <User className="text-primary size-4 md:size-5" />
                      </div>
                      <span className="text-sm font-medium md:text-base">{username}</span>
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={handleSwitchUser}
                      className="text-xs md:text-sm"
                    >
                      Switch User
                    </Button>
                  </div>
                )}
                <div className="space-y-3">
                  <Label>PIN</Label>
                  <div className="flex justify-center">
                    <InputOTP maxLength={4} value={pin} onChange={setPin}>
                      <InputOTPGroup className="gap-2 *:data-[slot=input-otp-slot]:rounded-md *:data-[slot=input-otp-slot]:border md:gap-2.5">
                        <InputOTPSlot index={0} className="size-10 text-lg md:size-12 md:text-xl" />
                        <InputOTPSlot index={1} className="size-10 text-lg md:size-12 md:text-xl" />
                        <InputOTPSlot index={2} className="size-10 text-lg md:size-12 md:text-xl" />
                        <InputOTPSlot index={3} className="size-10 text-lg md:size-12 md:text-xl" />
                      </InputOTPGroup>
                    </InputOTP>
                  </div>
                </div>
                <Button
                  type="submit"
                  className="w-full text-sm md:text-base"
                  disabled={loginMutation.isPending || pin.length !== 4 || username.trim() === ''}
                >
                  {loginMutation.isPending ? (
                    <Loader2 className="mr-1 size-3 animate-spin md:mr-2 md:size-4" />
                  ) : null}
                  Sign In
                </Button>
              </form>

              {!passkeyLoading && passkeySupported ? (
                <div className="mt-4 text-center">
                  <button
                    type="button"
                    onClick={() => setShowPinForm(false)}
                    className="text-muted-foreground hover:text-foreground text-sm hover:underline"
                  >
                    Use Passkey instead
                  </button>
                </div>
              ) : null}
            </>
          )}

          {/* Temporary debug button */}
          {!authStatus?.requiresSetup && (
            <div className="border-border mt-6 border-t pt-4">
              <Button
                type="button"
                variant="destructive"
                size="sm"
                className="w-full text-xs md:text-sm"
                onClick={handleDeleteAdmin}
                disabled={isDeleting}
              >
                {isDeleting ? (
                  <Loader2 className="mr-1 size-3 animate-spin md:mr-2 md:size-4" />
                ) : (
                  <Trash2 className="mr-1 size-3 md:mr-2 md:size-4" />
                )}
                Delete Admin (Debug)
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

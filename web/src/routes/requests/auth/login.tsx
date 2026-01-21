import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Loader2, Trash2, User, KeyRound } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { InputOTP, InputOTPGroup, InputOTPSlot } from '@/components/ui/input-otp'
import { usePortalLogin, useAuthStatus } from '@/hooks'
import { usePasskeySupport, usePasskeyLogin } from '@/hooks/portal'
import { usePortalAuthStore } from '@/stores'
import { deleteAdmin } from '@/api/auth'
import { toast } from 'sonner'

export function LoginPage() {
  const navigate = useNavigate()
  const { getPostLoginRedirect } = usePortalAuthStore()
  const loginMutation = usePortalLogin()
  const passkeyLoginMutation = usePasskeyLogin()
  const { data: authStatus, refetch: refetchAuthStatus } = useAuthStatus()
  const { isSupported: passkeySupported, isLoading: passkeyLoading } = usePasskeySupport()

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
    if (loginMutation.isPending) return

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
      }
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

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Welcome Back</CardTitle>
          <CardDescription>Sign in to your SlipStream account</CardDescription>
        </CardHeader>
        <CardContent>
          {passkeyLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="size-6 animate-spin text-muted-foreground" />
            </div>
          ) : shouldShowPasskeyLogin ? (
            <div className="space-y-6">
              <Button
                onClick={handlePasskeyLogin}
                disabled={passkeyLoginMutation.isPending}
                className="w-full text-sm md:text-base"
              >
                {passkeyLoginMutation.isPending ? (
                  <Loader2 className="size-3 md:size-4 mr-1 md:mr-2 animate-spin" />
                ) : (
                  <KeyRound className="size-3 md:size-4 mr-1 md:mr-2" />
                )}
                Sign in with Passkey
              </Button>

              <div className="text-center">
                <button
                  type="button"
                  onClick={() => setShowPinForm(true)}
                  className="text-sm text-muted-foreground hover:text-foreground hover:underline"
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
                  <div className="flex items-center justify-between p-2 md:p-3 rounded-lg border border-border bg-muted/50">
                    <div className="flex items-center gap-2 md:gap-3">
                      <div className="p-1.5 md:p-2 rounded-full bg-primary/10">
                        <User className="size-4 md:size-5 text-primary" />
                      </div>
                      <span className="font-medium text-sm md:text-base">{username}</span>
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
                    <InputOTP
                      maxLength={4}
                      value={pin}
                      onChange={setPin}
                    >
                      <InputOTPGroup className="gap-2 md:gap-2.5 *:data-[slot=input-otp-slot]:rounded-md *:data-[slot=input-otp-slot]:border">
                        <InputOTPSlot index={0} className="size-10 md:size-12 text-lg md:text-xl" />
                        <InputOTPSlot index={1} className="size-10 md:size-12 text-lg md:text-xl" />
                        <InputOTPSlot index={2} className="size-10 md:size-12 text-lg md:text-xl" />
                        <InputOTPSlot index={3} className="size-10 md:size-12 text-lg md:text-xl" />
                      </InputOTPGroup>
                    </InputOTP>
                  </div>
                </div>
                <Button
                  type="submit"
                  className="w-full text-sm md:text-base"
                  disabled={loginMutation.isPending || pin.length !== 4 || username.trim() === ''}
                >
                  {loginMutation.isPending && <Loader2 className="size-3 md:size-4 mr-1 md:mr-2 animate-spin" />}
                  Sign In
                </Button>
              </form>

              {!passkeyLoading && passkeySupported && (
                <div className="mt-4 text-center">
                  <button
                    type="button"
                    onClick={() => setShowPinForm(false)}
                    className="text-sm text-muted-foreground hover:text-foreground hover:underline"
                  >
                    Use Passkey instead
                  </button>
                </div>
              )}
            </>
          )}

          {/* Temporary debug button */}
          {!authStatus?.requiresSetup && (
            <div className="mt-6 pt-4 border-t border-border">
              <Button
                type="button"
                variant="destructive"
                size="sm"
                className="w-full text-xs md:text-sm"
                onClick={handleDeleteAdmin}
                disabled={isDeleting}
              >
                {isDeleting ? (
                  <Loader2 className="size-3 md:size-4 mr-1 md:mr-2 animate-spin" />
                ) : (
                  <Trash2 className="size-3 md:size-4 mr-1 md:mr-2" />
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

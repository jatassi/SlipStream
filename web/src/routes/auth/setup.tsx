import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { InputOTP, InputOTPGroup, InputOTPSlot } from '@/components/ui/input-otp'
import { useAdminSetup } from '@/hooks'
import { toast } from 'sonner'

export function SetupPage() {
  const navigate = useNavigate()
  const setupMutation = useAdminSetup()

  const [pin, setPin] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    if (pin.length !== 4) {
      toast.error('PIN must be exactly 4 digits')
      return
    }

    setupMutation.mutate(pin, {
      onSuccess: () => {
        toast.success('Administrator account created')
        navigate({ to: '/' })
      },
      onError: (error) => {
        toast.error('Setup failed', {
          description: error.message || 'Failed to create administrator account',
        })
      },
    })
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Welcome to SlipStream</CardTitle>
          <CardDescription>Create your administrator account to get started</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                type="text"
                value="Administrator"
                disabled
                className="bg-muted"
              />
            </div>
            <div className="space-y-3">
              <Label>PIN</Label>
              <div className="flex justify-center">
                <InputOTP
                  maxLength={4}
                  value={pin}
                  onChange={setPin}
                  autoFocus
                >
                  <InputOTPGroup className="gap-2.5 *:data-[slot=input-otp-slot]:rounded-md *:data-[slot=input-otp-slot]:border">
                    <InputOTPSlot index={0} className="size-12 text-xl" />
                    <InputOTPSlot index={1} className="size-12 text-xl" />
                    <InputOTPSlot index={2} className="size-12 text-xl" />
                    <InputOTPSlot index={3} className="size-12 text-xl" />
                  </InputOTPGroup>
                </InputOTP>
              </div>
              <p className="text-xs text-muted-foreground text-center">
                Choose a 4-digit PIN you'll remember
              </p>
            </div>
            <Button
              type="submit"
              className="w-full"
              disabled={setupMutation.isPending || pin.length !== 4}
            >
              {setupMutation.isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
              Create Administrator
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

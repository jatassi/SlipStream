import { useState, useCallback, useRef } from 'react'
import {
  usePasskeySupport,
  usePasskeyCredentials,
  useRegisterPasskey,
  useDeletePasskey,
  useUpdatePasskeyName,
} from '@/hooks/portal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { InputOTP, InputOTPGroup, InputOTPSlot } from '@/components/ui/input-otp'
import { KeyRound, Plus, Trash2, Pencil, Check, X, Loader2 } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'

export function PasskeyManager() {
  const [newPasskeyName, setNewPasskeyName] = useState('')
  const [pin, setPin] = useState('')
  const [isRegistering, setIsRegistering] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editingName, setEditingName] = useState('')
  const isSubmittingRef = useRef(false)

  const { isSupported, isLoading: isSupportLoading } = usePasskeySupport()
  const { data: credentials, isLoading } = usePasskeyCredentials()
  const registerPasskey = useRegisterPasskey()
  const deletePasskey = useDeletePasskey()
  const updateName = useUpdatePasskeyName()

  const handleRegister = useCallback(async (pinValue: string, nameValue: string) => {
    if (!nameValue.trim() || pinValue.length !== 4 || registerPasskey.isPending || isSubmittingRef.current) return
    isSubmittingRef.current = true

    try {
      await registerPasskey.mutateAsync({ pin: pinValue, name: nameValue })
      setNewPasskeyName('')
      setPin('')
      setIsRegistering(false)
    } catch {
      setPin('')
    } finally {
      isSubmittingRef.current = false
    }
  }, [registerPasskey])

  const handlePinChange = (value: string) => {
    setPin(value)
    if (value.length === 4 && newPasskeyName.trim()) {
      handleRegister(value, newPasskeyName)
    }
  }

  if (isSupportLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!isSupported) {
    return (
      <div className="rounded-lg border border-border p-4 text-muted-foreground">
        Passkeys require a secure connection (HTTPS).
      </div>
    )
  }

  const handleStartEdit = (id: string, currentName: string) => {
    setEditingId(id)
    setEditingName(currentName)
  }

  const handleSaveEdit = async () => {
    if (!editingId || !editingName.trim()) return
    await updateName.mutateAsync({ id: editingId, name: editingName })
    setEditingId(null)
    setEditingName('')
  }

  const handleCancelEdit = () => {
    setEditingId(null)
    setEditingName('')
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium">Passkeys</h3>
          <p className="text-sm text-muted-foreground">
            Use passkeys for faster, more secure sign-in
          </p>
        </div>
        {!isRegistering && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsRegistering(true)}
          >
            <Plus className="mr-2 h-4 w-4" />
            Add Passkey
          </Button>
        )}
      </div>

      {isRegistering && (
        <div className="rounded-lg border border-border p-4 space-y-4">
          <div className="space-y-2">
            <Label>Passkey Name</Label>
            <Input
              placeholder="e.g., MacBook Touch ID"
              value={newPasskeyName}
              onChange={(e) => setNewPasskeyName(e.target.value)}
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label>Enter PIN to confirm</Label>
            <div className="flex justify-center">
              {registerPasskey.isPending ? (
                <Loader2 className="h-10 w-10 animate-spin text-muted-foreground" />
              ) : (
                <InputOTP
                  maxLength={4}
                  value={pin}
                  onChange={handlePinChange}
                  disabled={!newPasskeyName.trim()}
                >
                  <InputOTPGroup className="gap-2 *:data-[slot=input-otp-slot]:rounded-md *:data-[slot=input-otp-slot]:border">
                    <InputOTPSlot index={0} className="size-10 text-lg" />
                    <InputOTPSlot index={1} className="size-10 text-lg" />
                    <InputOTPSlot index={2} className="size-10 text-lg" />
                    <InputOTPSlot index={3} className="size-10 text-lg" />
                  </InputOTPGroup>
                </InputOTP>
              )}
            </div>
          </div>
          <div className="flex justify-end">
            <Button
              variant="ghost"
              onClick={() => {
                setIsRegistering(false)
                setNewPasskeyName('')
                setPin('')
              }}
            >
              Cancel
            </Button>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : credentials?.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-8 text-center text-muted-foreground">
          No passkeys registered. Add one for faster, more secure login.
        </div>
      ) : (
        <div className="space-y-2">
          {credentials?.map((cred) => (
            <div
              key={cred.id}
              className="flex items-center justify-between rounded-lg border border-border p-3"
            >
              <div className="flex items-center gap-3">
                <KeyRound className="h-5 w-5 text-muted-foreground" />
                <div>
                  {editingId === cred.id ? (
                    <div className="flex items-center gap-2">
                      <Input
                        value={editingName}
                        onChange={(e) => setEditingName(e.target.value)}
                        className="h-8 w-48"
                        autoFocus
                      />
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={handleSaveEdit}
                        disabled={updateName.isPending}
                      >
                        <Check className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={handleCancelEdit}
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    </div>
                  ) : (
                    <>
                      <div className="font-medium">{cred.name}</div>
                      <div className="text-sm text-muted-foreground">
                        Created {formatDistanceToNow(new Date(cred.createdAt))} ago
                        {cred.lastUsedAt && (
                          <> · Last used {formatDistanceToNow(new Date(cred.lastUsedAt))} ago</>
                        )}
                      </div>
                    </>
                  )}
                </div>
              </div>
              {editingId !== cred.id && (
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleStartEdit(cred.id, cred.name)}
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => deletePasskey.mutate(cred.id)}
                    disabled={deletePasskey.isPending}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

import * as React from 'react'

import { OTPInput, OTPInputContext } from 'input-otp'
import { MinusIcon } from 'lucide-react'

import { cn } from '@/lib/utils'

const InputOTPMaskContext = React.createContext(false)

function InputOTP({
  className,
  containerClassName,
  mask,
  ...props
}: React.ComponentProps<typeof OTPInput> & {
  containerClassName?: string
  mask?: boolean
}) {
  return (
    <InputOTPMaskContext.Provider value={mask ?? false}>
      <OTPInput
        data-slot="input-otp"
        containerClassName={cn(
          'cn-input-otp flex items-center has-disabled:opacity-50',
          containerClassName,
        )}
        spellCheck={false}
        className={cn('disabled:cursor-not-allowed', className)}
        {...props}
      />
    </InputOTPMaskContext.Provider>
  )
}

function InputOTPGroup({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      data-slot="input-otp-group"
      className={cn(
        'has-aria-invalid:ring-destructive/20 dark:has-aria-invalid:ring-destructive/40 has-aria-invalid:border-destructive flex items-center rounded-lg has-aria-invalid:ring-[3px]',
        className,
      )}
      {...props}
    />
  )
}

const MASK_DELAY_MS = 300

function InputOTPSlot({
  index,
  className,
  ...props
}: React.ComponentProps<'div'> & {
  index: number
}) {
  const inputOTPContext = React.useContext(OTPInputContext)
  const mask = React.useContext(InputOTPMaskContext)
  const slot = inputOTPContext.slots[index]
  const { char, hasFakeCaret, isActive } = slot

  const [masked, setMasked] = React.useState(false)
  const [prevChar, setPrevChar] = React.useState(char)

  if (char !== prevChar) {
    setPrevChar(char)
    if (masked) { setMasked(false) }
  }

  React.useEffect(() => {
    if (!mask || !char) { return }
    const timer = setTimeout(() => setMasked(true), MASK_DELAY_MS)
    return () => clearTimeout(timer)
  }, [mask, char])

  const displayChar = mask && char && masked ? '\u25CF' : char

  return (
    <div
      data-slot="input-otp-slot"
      data-active={isActive}
      className={cn(
        'dark:bg-input/30 border-input data-[active=true]:border-ring data-[active=true]:ring-ring/50 data-[active=true]:aria-invalid:ring-destructive/20 dark:data-[active=true]:aria-invalid:ring-destructive/40 aria-invalid:border-destructive data-[active=true]:aria-invalid:border-destructive relative flex size-8 items-center justify-center border-y border-r text-sm transition-all outline-none first:rounded-l-lg first:border-l last:rounded-r-lg data-[active=true]:z-10 data-[active=true]:ring-[3px]',
        className,
      )}
      {...props}
    >
      {displayChar}
      {hasFakeCaret ? (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
          <div className="animate-caret-blink bg-foreground bg-foreground h-4 w-px duration-1000" />
        </div>
      ) : null}
    </div>
  )
}

function InputOTPSeparator({ ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      data-slot="input-otp-separator"
      className="flex items-center [&_svg:not([class*='size-'])]:size-4"
      role="separator"
      {...props}
    >
      <MinusIcon />
    </div>
  )
}

export { InputOTP, InputOTPGroup, InputOTPSeparator, InputOTPSlot }

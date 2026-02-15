import { cn } from '@/lib/utils'

type NetworkLogoProps = {
  logoUrl?: string
  network?: string
  className?: string
}

export function NetworkLogo({ logoUrl, network, className }: NetworkLogoProps) {
  if (!logoUrl) {
    return null
  }

  return (
    <div
      className={cn('rounded-md bg-black/60 px-1.5 py-1 backdrop-blur-sm', className)}
      title={network}
    >
      <img
        src={logoUrl}
        alt={network ?? 'Network'}
        className="h-4 w-auto max-w-full object-contain brightness-0 invert md:h-5"
      />
    </div>
  )
}

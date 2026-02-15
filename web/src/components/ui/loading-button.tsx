import type { VariantProps } from 'class-variance-authority'
import type { LucideIcon } from 'lucide-react'
import { Loader2 } from 'lucide-react'

import { cn } from '@/lib/utils'

import { Button, type buttonVariants } from './button'

type ButtonProps = React.ComponentProps<typeof Button>

type LoadingButtonProps = ButtonProps & {
  loading: boolean
  icon?: LucideIcon
  iconClassName?: string
}

function LoadingButton({ loading, icon: Icon, iconClassName, children, disabled, ...props }: LoadingButtonProps) {
  const isIconOnly = (props as VariantProps<typeof buttonVariants>).size?.toString().startsWith('icon')
  const hasTextContent = !isIconOnly && children

  let iconEl: React.ReactNode = null
  if (loading) {
    iconEl = <Loader2 className={cn('animate-spin', hasTextContent && 'mr-2', iconClassName)} />
  } else if (Icon) {
    iconEl = <Icon className={cn(hasTextContent && 'mr-2', iconClassName)} />
  }

  return (
    <Button disabled={disabled ?? loading} {...props}>
      {iconEl}
      {children}
    </Button>
  )
}

export { LoadingButton }

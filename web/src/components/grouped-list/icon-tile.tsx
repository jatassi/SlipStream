import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

export function IconTile({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <span className={cn('flex size-7 items-center justify-center rounded-[7px] text-white [&_svg]:size-4', className)}>
      {children}
    </span>
  )
}

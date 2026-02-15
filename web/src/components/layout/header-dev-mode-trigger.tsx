import { Hammer } from 'lucide-react'

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

export function HeaderDevModeTrigger() {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger className="text-muted-foreground inline-flex h-8 w-8 items-center justify-center rounded-md">
          <Hammer className="size-4" />
        </TooltipTrigger>
        <TooltipContent>Enable developer mode</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

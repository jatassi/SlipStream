import { Hammer, LayoutTemplate, Loader2 } from 'lucide-react'

import { Label } from '@/components/ui/label'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'

type HeaderDevModePopoverProps = {
  devModeSwitching: boolean
  globalLoading: boolean
  onGlobalLoadingChange: (checked: boolean) => void
}

export function HeaderDevModePopover({
  devModeSwitching,
  globalLoading,
  onGlobalLoadingChange,
}: HeaderDevModePopoverProps) {
  return (
    <Popover>
      <PopoverTrigger
        className={cn(
          'inline-flex h-8 w-8 items-center justify-center rounded-md transition-colors',
          'text-amber-500 hover:bg-amber-600/20',
        )}
      >
        {devModeSwitching ? (
          <Loader2 className="size-4 animate-spin" />
        ) : (
          <Hammer className="size-4" />
        )}
      </PopoverTrigger>
      <PopoverContent align="end" className="w-56 gap-0 p-0">
        <div className="border-border border-b px-3 py-2">
          <span className="text-muted-foreground text-xs font-medium">Developer Tools</span>
        </div>
        <div className="space-y-1 p-2">
          <Label
            htmlFor="force-loading-toggle"
            className="hover:bg-accent flex cursor-pointer items-center gap-2.5 rounded-md px-2 py-1.5 text-sm transition-colors"
          >
            <LayoutTemplate className="text-muted-foreground size-4 shrink-0" />
            <span className="flex-1">Force Loading</span>
            <Switch
              id="force-loading-toggle"
              checked={globalLoading}
              onCheckedChange={onGlobalLoadingChange}
              size="sm"
            />
          </Label>
        </div>
      </PopoverContent>
    </Popover>
  )
}

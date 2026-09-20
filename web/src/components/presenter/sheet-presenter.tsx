import type { ReactNode } from 'react'

import { Drawer } from 'vaul'

import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { useViewport } from '@/hooks/use-viewport'
import { cn } from '@/lib/utils'

import './sheet-presenter.css'

export type SheetPresenterProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  children: ReactNode
  footer?: ReactNode
}

export function SheetPresenter({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
}: SheetPresenterProps) {
  const shell = useViewport()

  if (shell === 'phone') {
    return (
      <SheetSurface
        open={open}
        onOpenChange={onOpenChange}
        title={title}
        description={description}
        footer={footer}
      >
        {children}
      </SheetSurface>
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription className={description === undefined ? 'sr-only' : undefined}>
            {description ?? title}
          </DialogDescription>
        </DialogHeader>
        <DialogBody>{children}</DialogBody>
        {footer === undefined ? null : <DialogFooter>{footer}</DialogFooter>}
      </DialogContent>
    </Dialog>
  )
}

function SheetSurface({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
}: SheetPresenterProps) {
  return (
    <Drawer.Root open={open} onOpenChange={onOpenChange} repositionInputs={false}>
      <Drawer.Portal>
        <Drawer.Overlay className="sheet-presenter-scrim" />
        <Drawer.Content className="sheet-presenter">
          <Drawer.Handle className="my-3 shrink-0" />
          <div className="px-4 pb-2 text-center">
            <Drawer.Title className="text-title font-semibold">{title}</Drawer.Title>
            <Drawer.Description
              className={cn(
                description === undefined
                  ? 'sr-only'
                  : 'text-footnote text-muted-foreground mt-0.5',
              )}
            >
              {description ?? title}
            </Drawer.Description>
          </div>
          <div className="scroll min-h-0 flex-1 px-4 pb-2">{children}</div>
          {footer === undefined ? null : (
            <div className="safe-bottom shrink-0 px-4 pt-2 pb-4">{footer}</div>
          )}
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  )
}

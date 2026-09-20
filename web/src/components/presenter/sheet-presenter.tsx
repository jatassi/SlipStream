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
  description?: ReactNode
  children: ReactNode
  footer?: ReactNode
  /** Extra classes for the wide dialog, for forms that need more than the default width. */
  wideClassName?: string
  /** Set when this presenter is rendered inside another presenter's children. */
  nested?: boolean
}

export function SheetPresenter({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
  wideClassName,
  nested = false,
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
        nested={nested}
      >
        {children}
      </SheetSurface>
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={cn('sm:max-w-md', wideClassName)}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription className={description === undefined ? 'sr-only' : undefined}>
            {description ?? title}
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="presented-form">{children}</DialogBody>
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
  nested,
}: SheetPresenterProps) {
  const Root = nested === true ? Drawer.NestedRoot : Drawer.Root

  return (
    <Root open={open} onOpenChange={onOpenChange} repositionInputs={false}>
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
          <div className="scroll presented-form min-h-0 flex-1 px-4 pb-2">{children}</div>
          {footer === undefined ? null : (
            <div className="safe-bottom shrink-0 px-4 pt-2 pb-4">{footer}</div>
          )}
        </Drawer.Content>
      </Drawer.Portal>
    </Root>
  )
}

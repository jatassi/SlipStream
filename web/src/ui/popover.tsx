import { Popover as BasePopover } from "@base-ui/react/popover";
import type { ReactElement, ReactNode } from "react";

import { usePortalDensity } from "./portal-density";

const opening = {
  press: {},
  hover: { openOnHover: true, closeDelay: 160 },
} as const;

export type PopoverProps = {
  trigger: ReactElement;
  label: string;
  opensOn?: keyof typeof opening;
  children: ReactNode;
};

const popup =
  "w-[340px] max-w-(--available-width) origin-(--transform-origin) rounded-card bg-popover p-4 text-(length:--body-size) leading-(--body-line-height) text-ink-1 shadow-float outline-none backdrop-blur-[30px] backdrop-saturate-[1.7] transition-[translate,scale,opacity] duration-(--duration-menu) ease-standard data-starting-style:opacity-0 data-ending-style:opacity-0 motion-safe:data-starting-style:translate-y-2 motion-safe:data-starting-style:scale-[.985] motion-safe:data-ending-style:translate-y-2 motion-safe:data-ending-style:scale-[.985]";

export function Popover({ trigger, label, opensOn = "press", children }: PopoverProps) {
  const { density, probe } = usePortalDensity();
  return (
    <>
      {probe}
      <BasePopover.Root>
        <BasePopover.Trigger render={trigger} {...opening[opensOn]} />
        <BasePopover.Portal data-density={density}>
          <BasePopover.Positioner sideOffset={6} collisionPadding={12}>
            <BasePopover.Popup aria-label={label} className={popup}>
              {children}
            </BasePopover.Popup>
          </BasePopover.Positioner>
        </BasePopover.Portal>
      </BasePopover.Root>
    </>
  );
}

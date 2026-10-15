import { Popover as BasePopover } from "@base-ui/react/popover";
import type { ReactElement, ReactNode } from "react";

import { floatingPanel } from "./floating";
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

const popup = `w-[340px] max-w-(--available-width) p-4 text-(length:--body-size) leading-(--body-line-height) ${floatingPanel}`;

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

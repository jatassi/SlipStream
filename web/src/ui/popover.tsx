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
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  /** 340px for a detail, 400px for a list of rows such as the bell's. */
  width?: keyof typeof widths;
  children: ReactNode;
};

const widths = { 340: "w-[340px]", 400: "w-[400px]" } as const;

const popupLook = `max-w-(--available-width) p-4 text-(length:--body-size) leading-(--body-line-height) ${floatingPanel}`;

export const detailPopup = `${widths[340]} ${popupLook}`;

export function Popover({
  trigger,
  label,
  opensOn = "press",
  width = 340,
  children,
  ...root
}: PopoverProps) {
  const { density, probe } = usePortalDensity();
  return (
    <>
      {probe}
      <BasePopover.Root {...root}>
        <BasePopover.Trigger render={trigger} {...opening[opensOn]} />
        <BasePopover.Portal data-density={density}>
          <BasePopover.Positioner sideOffset={6} collisionPadding={12}>
            <BasePopover.Popup aria-label={label} className={`${widths[width]} ${popupLook}`}>
              {children}
            </BasePopover.Popup>
          </BasePopover.Positioner>
        </BasePopover.Portal>
      </BasePopover.Root>
    </>
  );
}

import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import type { ReactElement } from "react";

export type TooltipProps = {
  label: string;
  count?: number;
  side?: "top" | "right" | "bottom" | "left";
  children: ReactElement;
};

const popup =
  "flex items-center gap-2 rounded-item bg-popover px-2.5 py-1.5 text-meta font-medium text-ink-1 shadow-float backdrop-blur-[30px] backdrop-saturate-[1.7] transition-opacity duration-(--duration-menu) ease-standard data-instant:duration-0 data-starting-style:opacity-0 data-ending-style:opacity-0";

// Base UI tooltips are visual only, so the trigger carries the same words as its name.
export function Tooltip({ label, count, side = "top", children }: TooltipProps) {
  return (
    <BaseTooltip.Root>
      <BaseTooltip.Trigger
        render={children}
        aria-label={count === undefined ? label : `${label}, ${count}`}
      />
      <BaseTooltip.Portal>
        <BaseTooltip.Positioner side={side} sideOffset={6} collisionPadding={12}>
          <BaseTooltip.Popup className={popup}>
            <span>{label}</span>
            {count !== undefined && <span className="text-ink-3 tabular-nums">{count}</span>}
          </BaseTooltip.Popup>
        </BaseTooltip.Positioner>
      </BaseTooltip.Portal>
    </BaseTooltip.Root>
  );
}

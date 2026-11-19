import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import type { ReactElement } from "react";

import { type AlertTone, dotOf, dotWord } from "./count";
import { labelPopup } from "./floating";

export type TooltipProps = {
  label: string;
  count?: number;
  alert?: AlertTone;
  side?: "top" | "right" | "bottom" | "left";
  /** The trigger's accessible name when it differs from the tooltip's words. */
  name?: string;
  children: ReactElement;
};

const popup = `flex items-center gap-2 ${labelPopup}`;

// Base UI tooltips are visual only, so the trigger carries the same words as its name.
export function Tooltip({
  label,
  count,
  alert,
  side = "top",
  name: given,
  children,
}: TooltipProps) {
  const name =
    given ??
    [label, count, alert === undefined ? undefined : dotWord(alert)]
      .filter((part) => part !== undefined)
      .join(", ");
  return (
    <BaseTooltip.Root>
      <BaseTooltip.Trigger render={children} aria-label={name} />
      <BaseTooltip.Portal>
        <BaseTooltip.Positioner side={side} sideOffset={6} collisionPadding={12}>
          <BaseTooltip.Popup className={popup}>
            <span>{label}</span>
            {count !== undefined && <span className="text-ink-3 tabular-nums">{count}</span>}
            {alert !== undefined && dotOf(alert)}
          </BaseTooltip.Popup>
        </BaseTooltip.Positioner>
      </BaseTooltip.Portal>
    </BaseTooltip.Root>
  );
}

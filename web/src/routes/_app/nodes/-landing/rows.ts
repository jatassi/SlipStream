import type { KeyboardEvent } from "react";

import { moveBetweenRows, rowGroup, rowTarget } from "../../../../ui/row-keys";
import "../../../../ui/rows.css";

export const rosterGroup = { [rowGroup]: "" };

export const rowShell =
  "relative first:rounded-t-card last:rounded-b-card not-first:before:absolute not-first:before:inset-x-0 not-first:before:top-0 not-first:before:h-px not-first:before:bg-hairline in-data-[density=touch]:not-first:before:left-4";

export const rowWash =
  "rounded-[inherit] transition-colors duration-350 ease-standard hover:bg-(--row-wash) focus-visible:bg-(--row-wash)";

/** Up and down arrows, Home and End move between the Roster's rows. */
export const rowKeys = {
  [rowTarget]: "",
  onKeyDown: (event: KeyboardEvent<HTMLElement>) => {
    moveBetweenRows(event);
  },
};

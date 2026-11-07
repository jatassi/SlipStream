import type { ReactNode } from "react";

import { segmentedOption, segmentedTracks } from "./segmented";
import { useShell } from "./shell-context";

/** A link in SegmentedLinks. The router marks the current one `aria-current="page"`. */
export const segmentedLink = `${segmentedOption} cursor-pointer hover:text-ink-1 aria-[current=page]:bg-selected aria-[current=page]:text-ink-1 aria-[current=page]:shadow-[0_1px_3px_rgba(0,0,0,.28)]`;

/**
 * The Segmented control's look over links, for a choice where each option is
 * its own address. A link can end in Segmented's SegmentCount.
 */
export function SegmentedLinks({ label, children }: { label: string; children: ReactNode }) {
  return (
    <nav aria-label={label} className={segmentedTracks[useShell()]}>
      {children}
    </nav>
  );
}

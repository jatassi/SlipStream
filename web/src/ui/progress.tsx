import { Progress } from "@base-ui/react/progress";
import type { ReactNode } from "react";

import { Data } from "./data";

/** Live while the work moves, settled while it's paused or its numbers are an old sample. */
export type Motion = "live" | "settled";

const looks: { readonly [M in Motion]: { readonly track: string; readonly fill: string } } = {
  live: {
    track: "bg-[hsl(var(--h)_60%_80%/.14)] ring-1 ring-[hsl(var(--h)_90%_65%/.32)] ring-inset",
    fill: "relative overflow-hidden bg-linear-to-r from-glow to-solid shadow-live-glow transition-[width] duration-(--duration-live-tick) ease-linear after:absolute after:inset-0 after:animate-sheen after:bg-linear-to-r after:from-transparent after:via-[rgba(255,255,255,.55)] after:to-transparent motion-reduce:transition-none motion-reduce:after:animate-none",
  },
  settled: { track: "bg-fill", fill: "bg-ink-2" },
};

/** A whole percent of done over total, floored so it reads 100 only when done. `empty` is the percent with nothing to do. */
export const percentOf = (done: number, total: number, empty: 0 | 100): number =>
  total === 0 ? empty : Math.floor((done / total) * 100);

/** Work that is moving, as a 6px bar whichever its motion, so a pause never shifts the row. */
export function ProgressBar({
  label,
  value,
  motion,
}: {
  label: string;
  value: number;
  motion: Motion;
}) {
  // Base UI appends a hidden "x" to the bar's children, which leaks into the
  // text of the row around it, so the bar renders its track as its only child.
  const track = (
    <Progress.Track className={`h-1.5 overflow-hidden rounded-capsule ${looks[motion].track}`}>
      <Progress.Indicator className={`rounded-capsule ${looks[motion].fill}`} />
    </Progress.Track>
  );
  return (
    <Progress.Root
      value={value}
      aria-label={label}
      data-motion={motion}
      className="w-full"
      render={(props) => <div {...props}>{track}</div>}
    />
  );
}

const spans = { cell: "max-w-60", line: "max-w-90", fill: "" } as const;

/**
 * A bar under its amount and a note: the amount on the left in ink-1, the
 * percent unless the caller names it, and the note on the right in ink-3,
 * such as the pace, the time left or when it started. It is at most 240px
 * wide in a table cell and 360px in a line.
 */
export function ProgressCell({
  label,
  value,
  motion,
  amount,
  note,
  span = "cell",
}: {
  label: string;
  value: number;
  motion: Motion;
  amount?: ReactNode;
  note?: ReactNode;
  span?: keyof typeof spans;
}) {
  return (
    <span className={`flex w-full min-w-0 flex-col gap-1.5 ${spans[span]}`}>
      <span className="flex items-baseline justify-between gap-3 text-meta text-ink-3">
        <span className="shrink-0 whitespace-nowrap text-ink-1">
          {amount ?? <Data>{value}%</Data>}
        </span>
        {note !== undefined && <span className="min-w-0 truncate text-right">{note}</span>}
      </span>
      <ProgressBar label={label} value={value} motion={motion} />
    </span>
  );
}

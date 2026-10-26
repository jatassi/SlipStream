import type { GridLayout } from "./grid-layout";

export type Step = "left" | "right" | "up" | "down" | "first" | "last";

const steps: Readonly<Record<string, Step>> = {
  ArrowLeft: "left",
  ArrowRight: "right",
  ArrowUp: "up",
  ArrowDown: "down",
  Home: "first",
  End: "last",
};

export const stepOf = (key: string): Step | undefined => steps[key];

const clamp = (index: number, count: number) => Math.min(Math.max(index, 0), count - 1);

// The poster row `by` rows of posters away, skipping section headings.
function posterRow({ rows }: GridLayout, from: number, by: -1 | 1) {
  for (let at = from + by; at >= 0 && at < rows.length; at += by) {
    const row = rows[at];
    if (row?.kind === "posters") {
      return row;
    }
  }
  return undefined;
}

/**
 * Where focus goes in the grid: left and right step one Work, up and down
 * keep the column into the next row of posters, which may sit past a
 * section heading or hold fewer Works.
 */
export function gridMove(layout: GridLayout, index: number, step: Step, count: number): number {
  if (step === "left" || step === "right") {
    return clamp(index + (step === "left" ? -1 : 1), count);
  }
  if (step === "first" || step === "last") {
    return tableMove(index, step, count);
  }
  const at = layout.rowOf(index);
  const row = layout.rows[at];
  if (row?.kind !== "posters") {
    return clamp(index, count);
  }
  const next = posterRow(layout, at, step === "up" ? -1 : 1);
  if (next === undefined) {
    return index;
  }
  return next.first + Math.min(index - row.first, next.ids.length - 1);
}

/** Where focus goes in the table: one row up or down, or to either end. */
export function tableMove(index: number, step: Step, count: number): number {
  if (step === "first") {
    return 0;
  }
  if (step === "last") {
    return count - 1;
  }
  if (step === "up" || step === "down") {
    return clamp(index + (step === "up" ? -1 : 1), count);
  }
  return index;
}

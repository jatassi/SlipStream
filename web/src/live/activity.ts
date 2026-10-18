// When each kind of work the Diagram's machines show last reported, by the
// page's clock, outside React Query like the traffic ring: the live loop
// reads it every frame.

/** A search started or finished, or a Work's artwork and details were fetched. */
export type Work = "search" | "fetch";

const last = new Map<Work, number>();

export function noteWork(work: Work, at = performance.now()): void {
  last.set(work, at);
}

/** When the work last reported, or undefined when it hasn't since the page loaded. */
export const lastWork = (work: Work): number | undefined => last.get(work);

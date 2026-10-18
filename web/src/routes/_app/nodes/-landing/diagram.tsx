import type { Landing } from "./view";

const waiting = (count: number) =>
  count === 1
    ? "The Diagram of your Node is on its way. The Roster lists it."
    : `The Diagram of your ${count} Nodes is on its way. The Roster lists the same Nodes.`;

/**
 * Where the Diagram draws the Nodes from the same reading the Roster lists.
 * Below 1000px it is look-only: it takes no drags.
 */
export function Diagram({ landing, takesDrags }: { landing: Landing; takesDrags: boolean }) {
  return (
    <figure
      aria-label="Diagram"
      data-takes-drags={takesDrags}
      className="flex min-h-[420px] flex-col items-center justify-center gap-2 rounded-card bg-glass p-6 text-center ring-1 ring-hairline ring-inset"
    >
      <figcaption className="max-w-[440px] text-ink-2">{waiting(landing.nodes.length)}</figcaption>
    </figure>
  );
}

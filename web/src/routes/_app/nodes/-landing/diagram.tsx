import type { Landing } from "./view";

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
      <figcaption className="max-w-[440px] text-ink-2">
        The Diagram of{" "}
        {landing.nodes.length === 1 ? "your Node" : `your ${landing.nodes.length} Nodes`} is on its
        way. The Roster lists the same Nodes.
      </figcaption>
    </figure>
  );
}

import type { Dragging } from "./drag";

/** What follows the pointer while a chip is dragged: the Role, the plan's word for the drop, and its note. */
export function Ghost({ dragging }: { dragging: Dragging }) {
  const { gesture } = dragging;
  return (
    <div
      aria-hidden="true"
      data-verdict={gesture?.verdict}
      className="pointer-events-none fixed z-50 -translate-x-1/2 translate-y-4 rounded-card bg-popover px-3 py-1.5 text-meta text-ink-1 shadow-float ring-1 ring-hairline data-[verdict=allow]:ring-ok data-[verdict=block]:ring-fail data-[verdict=warn]:ring-warn"
      style={{ left: dragging.at[0], top: dragging.at[1] }}
    >
      <span>{dragging.add ? `Add ${dragging.label}` : `Move ${dragging.label}`}</span>
      {gesture !== undefined && (
        <span className="ml-2 text-ink-2" data-word="">
          {gesture.word}
        </span>
      )}
      {gesture?.note !== undefined && <span className="block text-ink-3">{gesture.note}</span>}
    </div>
  );
}

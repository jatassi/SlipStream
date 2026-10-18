import type { Dragging } from "./drag";

/**
 * What follows the pointer while a chip is dragged: the plan's word for the
 * drop, after the gesture itself when the word doesn't name the Role, and the
 * word's note.
 */
export function Ghost({ dragging }: { dragging: Dragging }) {
  const { gesture, label } = dragging;
  const named = gesture?.word.includes(label) === true;
  return (
    <div
      aria-hidden="true"
      data-verdict={gesture?.verdict}
      className="pointer-events-none fixed z-50 -translate-x-1/2 translate-y-4 rounded-card bg-popover px-3 py-1.5 text-meta text-ink-1 shadow-float ring-1 ring-hairline data-[verdict=allow]:ring-ok data-[verdict=block]:ring-fail data-[verdict=warn]:ring-warn"
      style={{ left: dragging.at[0], top: dragging.at[1] }}
    >
      {!named && <span className="mr-2">{dragging.add ? `Add ${label}` : `Move ${label}`}</span>}
      {gesture !== undefined && (
        <span className={named ? "" : "text-ink-2"} data-word="">
          {gesture.word}
        </span>
      )}
      {gesture?.note !== undefined && <span className="block text-ink-3">{gesture.note}</span>}
    </div>
  );
}

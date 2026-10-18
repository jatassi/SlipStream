import { MotionConfig, Reorder } from "motion/react";
import { type ReactNode, useEffect, useId, useRef, useState } from "react";

import { type Control, type ListState, RankedListRow } from "./ranked-list-row";
import {
  type RankedRow,
  type Rows,
  dropRow,
  moveSentence,
  ranksOf,
  stepRow,
  tieRow,
} from "./ranked-rows";

export type { RankedRow } from "./ranked-rows";
export { settle, tiersOf } from "./ranked-rows";

type RankedListProps<T> = {
  readonly label: string;
  readonly caption: ReactNode;
  readonly rows: Rows<T>;
  /** The whole new order, after a drag, a Move button, an arrow key or a tie. */
  readonly onChange: (rows: RankedRow<T>[]) => void;
  /** An item's name, for its buttons and what a move announces. */
  readonly name: (value: T) => string;
  readonly renderRow: (value: T, rank: number) => ReactNode;
  readonly renderTrailing?: (value: T) => ReactNode;
  /** Offers "Tie with the one above" on each item that has an item above it. */
  readonly ties?: boolean;
};

type Drag = { readonly id: string; readonly order: readonly string[] };

// Motion can end a drag before the render of its last reorder, so the drag
// is read back from a ref.
function useDrag<T>(rows: Rows<T>, commit: (next: RankedRow<T>[], id: string) => void) {
  const [drag, setDrag] = useState<Drag | null>(null);
  const dragged = useRef<Drag | null>(null);
  const set = (next: Drag | null) => {
    dragged.current = next;
    setDrag(next);
  };
  return {
    drag,
    start: (id: string) => {
      set({ id, order: rows.map((row) => row.id) });
    },
    reorder: (order: string[]) => {
      if (dragged.current !== null) {
        set({ ...dragged.current, order });
      }
    },
    end: () => {
      const ended = dragged.current;
      set(null);
      if (ended !== null) {
        commit(dropRow(rows, ended.order, ended.id), ended.id);
      }
    },
  };
}

// A step re-renders the row in its new place; its control keeps focus, or
// the handle does once that control is disabled.
function useRefocus() {
  const list = useRef<HTMLOListElement>(null);
  const target = useRef<{ id: string; control: Control } | null>(null);
  useEffect(() => {
    const wanted = target.current;
    target.current = null;
    if (wanted === null) {
      return;
    }
    const row = list.current?.querySelector(`[data-row="${CSS.escape(wanted.id)}"]`);
    const control = row?.querySelector<HTMLElement>(`[data-control="${wanted.control}"]`);
    const handle = row?.querySelector<HTMLElement>('[data-control="handle"]');
    (control?.hasAttribute("disabled") === false ? control : handle)?.focus();
  });
  return {
    list,
    request: (id: string, control: Control) => {
      target.current = { id, control };
    },
  };
}

const nameWith =
  <T,>(name: (value: T) => string) =>
  (row: RankedRow<T>) =>
    row.kind === "item" ? name(row.value) : row.label;

function useRankedList<T>(props: RankedListProps<T>, hintId: string) {
  const { rows, onChange, ties = false } = props;
  const [said, setSaid] = useState("");
  const nameOf = nameWith(props.name);
  const commit = (next: RankedRow<T>[], id: string) => {
    onChange(next);
    setSaid(moveSentence(next, id, nameOf));
  };
  const { drag, start, reorder, end } = useDrag(rows, commit);
  const { list, request } = useRefocus();
  const settled = drag === null ? rows : dropRow(rows, drag.order, drag.id);
  const state: ListState<T> = {
    rows,
    ranks: ranksOf(settled),
    dragging: drag?.id,
    hintId,
    ties,
    nameOf,
    renderRow: props.renderRow,
    renderTrailing: props.renderTrailing,
    step: (id, by, control) => {
      request(id, control);
      commit(stepRow(rows, id, by), id);
    },
    tie: (id, tied) => {
      onChange(tieRow(rows, id, tied));
    },
    dragStart: start,
    dragEnd: end,
  };
  const byId = new Map(rows.map((row) => [row.id, row]));
  const order = drag?.order ?? rows.map((row) => row.id);
  const shown = order.flatMap((id) => byId.get(id) ?? []);
  return { state, order, shown, said, list, reorder };
}

/**
 * A list whose order is its meaning. Each item has a rank numeral (tied items
 * share one), a drag handle that the arrow keys also move, and Move up and
 * Move down, which the Touch density gathers with the tie into one row
 * action. A marker, such as the cutoff, moves the same way and sits only
 * between tiers. Each move is announced.
 */
export function RankedList<T>(props: RankedListProps<T>) {
  const captionId = useId();
  const hintId = useId();
  const { state, order, shown, said, list, reorder } = useRankedList(props, hintId);
  return (
    <MotionConfig reducedMotion="user">
      <div className="flex flex-col gap-3">
        <p id={captionId} className="text-meta text-ink-3">
          {props.caption}
        </p>
        <span id={hintId} hidden>
          Press the up and down arrow keys to move it.
        </span>
        <Reorder.Group
          as="ol"
          ref={list}
          axis="y"
          values={[...order]}
          onReorder={reorder}
          aria-label={props.label}
          aria-describedby={captionId}
          className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
        >
          {shown.map((row) => (
            <RankedListRow key={row.id} row={row} list={state} />
          ))}
        </Reorder.Group>
        <p aria-live="polite" className="sr-only">
          {said}
        </p>
      </div>
    </MotionConfig>
  );
}

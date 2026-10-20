import { type KeyboardEvent, useEffect, useRef } from "react";

import { type Move, moveIndex } from "../ui/move-index";
import { rowTarget } from "../ui/row-keys";
import type { Selection } from "../ui/table";
import type { Row } from "./model";
import { type CellProps, WorkCell } from "./row";

const moves: Readonly<Record<string, Move>> = {
  ArrowDown: "next",
  ArrowUp: "prev",
  Home: "first",
  End: "last",
  PageDown: "pageDown",
  PageUp: "pageUp",
};

const pageRows = 10;

/**
 * The arrow keys, Home, End and the page keys on a row move the selection
 * through the rows as shown, and focus follows once that row is drawn.
 */
export function useStepper(
  rows: readonly Row[],
  selection: Selection<Row> | undefined,
  reveal: (index: number) => void,
): (event: KeyboardEvent<HTMLElement>) => void {
  const focusing = useRef<string>(undefined);
  useEffect(() => {
    const id = focusing.current;
    const target =
      id === undefined
        ? null
        : document.querySelector<HTMLElement>(`[data-torrent-id="${id}"] [${rowTarget}]`);
    if (target !== null) {
      target.focus({ preventScroll: true });
      focusing.current = undefined;
    }
  });
  return (event) => {
    const move = moves[event.key];
    if (selection === undefined || move === undefined) {
      return;
    }
    event.preventDefault();
    const from = rows.findIndex((row) => row.id === selection.selected);
    const next = rows[moveIndex(from, move, rows.length, pageRows)];
    if (next === undefined || next.id === selection.selected) {
      return;
    }
    reveal(rows.indexOf(next));
    selection.onSelect(next);
    focusing.current = next.id;
  };
}

export function WorkTarget({
  props,
  selection,
  onStep,
}: {
  props: CellProps;
  selection: Selection<Row>;
  onStep: ((event: KeyboardEvent<HTMLElement>) => void) | undefined;
}) {
  const selected = selection.selected === props.row.id;
  return (
    <button
      type="button"
      aria-pressed={selected}
      {...{ [rowTarget]: "" }}
      onClick={() => {
        selection.onSelect(props.row);
      }}
      onKeyDown={onStep}
      className="w-full min-w-0 text-left after:absolute after:inset-0 focus-visible:shadow-none focus-visible:outline-none"
    >
      {selected && (
        <span
          aria-hidden="true"
          className="absolute inset-y-2 left-0 w-[3px] rounded-capsule bg-solid shadow-live-glow"
        />
      )}
      <WorkCell {...props} />
    </button>
  );
}

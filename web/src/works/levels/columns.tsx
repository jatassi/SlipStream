import type { CSSProperties } from "react";

import { StateButton } from "../../ui/state-square";
import { type CellPopoverControls, cellTrigger } from "../../ui/cell-popover";
import { anyCut } from "../page/matrix-model";
import { type PoppedCell, cellLabel } from "./cell-detail";
import type { Column, ItemView, LevelTarget } from "./model";

const squareWidth = 42;

/** A column's width: its Targets' squares side by side, or a narrow one while the level has seen none of it. */
const widthOf = (column: Column, targets: readonly LevelTarget[]): CSSProperties => ({
  width: column.seen ? targets.length * squareWidth : 96,
});

/** The names over a level's squares: each Edition, then its Targets in Profile order. */
export function ColumnNames({
  columns,
  targets,
}: {
  columns: readonly Column[];
  targets: readonly LevelTarget[];
}) {
  return (
    <span aria-hidden="true" className="flex shrink-0 gap-3 text-label text-ink-3">
      {columns.map((column) => (
        <span
          key={column.edition}
          style={widthOf(column, targets)}
          className="flex flex-col justify-end"
        >
          {column.edition !== anyCut && <span className="truncate">{column.name}</span>}
          {column.seen ? (
            <span className="flex">
              {targets.map((target) => (
                <span key={target.id} title={target.name} className="w-10.5 truncate text-center">
                  {target.name}
                </span>
              ))}
            </span>
          ) : (
            <span>Not seen yet</span>
          )}
        </span>
      ))}
    </span>
  );
}

/** A cut this Item hasn't been seen in, while others in its level have: one dotted line. */
function Unseen({ code, column, style }: { code: string; column: Column; style: CSSProperties }) {
  return (
    <span style={style} className="flex h-7.5 items-center px-3">
      <span aria-hidden="true" className="w-full border-t-2 border-dotted border-ink-4" />
      <span className="sr-only">
        {code} {column.name}, not seen yet
      </span>
    </span>
  );
}

/** An Item's squares grouped by Edition. A cut it hasn't been seen in reads as a dotted line. */
export function Squares({
  view,
  columns,
  targets,
  controls,
}: {
  view: ItemView;
  columns: readonly Column[];
  targets: readonly LevelTarget[];
  controls: CellPopoverControls<PoppedCell>;
}) {
  return (
    <span className="flex shrink-0 gap-3">
      {columns.map((column, index) => {
        const group = view.groups[index];
        const style = widthOf(column, targets);
        if (!column.seen) {
          return <span key={column.edition} style={style} />;
        }
        if (group === undefined || group.kind === "unseen") {
          return <Unseen key={column.edition} code={view.code} column={column} style={style} />;
        }
        return (
          <span key={column.edition} style={style} className="flex">
            {targets.map((target, at) => {
              const cell = group.cells[at];
              if (cell === undefined) {
                return <span key={target.id} className="w-10.5" />;
              }
              const popped = { code: view.code, column, target, cell };
              return (
                <StateButton
                  key={target.id}
                  state={cell.state}
                  label={cellLabel(popped)}
                  quiet={view.quiet || view.withdrawn}
                  {...cellTrigger(controls, popped)}
                />
              );
            })}
          </span>
        );
      })}
    </span>
  );
}

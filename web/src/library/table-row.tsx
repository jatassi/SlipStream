import { Link } from "@tanstack/react-router";
import { memo } from "react";

import { Checkbox } from "../ui/checkbox";
import { Poster } from "../ui/poster";
import type { ColumnDef, Names } from "./columns";
import { type CellHandlers, focusTarget } from "./picking";
import type { LibraryWork } from "./rows";
import { leadFact } from "./values";
import "../ui/rows.css";

const rowLook =
  "absolute inset-x-0 top-0 min-h-(--row-height-two) text-(length:--body-size) leading-(--body-line-height) text-ink-1 shadow-[inset_0_1px_0_var(--color-hairline)] transition-colors duration-350 ease-standard hover:bg-(--row-wash) has-focus-visible:bg-(--row-wash) has-focus-visible:outline-2 has-focus-visible:-outline-offset-2 has-focus-visible:outline-solid";

type Cell = {
  readonly work: LibraryWork;
  readonly tabbable: boolean;
  readonly handlers: CellHandlers;
};

function TitleCell({ work, tabbable, handlers }: Cell) {
  const fact = leadFact(work);
  return (
    <span className="flex min-w-0 items-center gap-3 py-1.5">
      <Poster url={work.poster} />
      <span className="flex min-w-0 flex-col gap-0.5">
        <Link
          to="/works/$workId"
          params={{ workId: work.id }}
          {...{ [focusTarget]: "" }}
          tabIndex={tabbable ? 0 : -1}
          onClick={(event) => {
            handlers.onClick(event, work.id);
          }}
          onKeyDown={(event) => {
            handlers.onKeyDown(event, work.id);
          }}
          onFocus={() => {
            handlers.onFocus(work.id);
          }}
          className="truncate font-semibold outline-none"
        >
          {work.label}
        </Link>
        {fact !== undefined && <span className="truncate text-meta text-ink-3">{fact}</span>}
      </span>
    </span>
  );
}

export type RowProps = Cell & {
  readonly index: number;
  readonly start: number;
  readonly measure: (element: HTMLElement | null) => void;
  readonly columns: readonly ColumnDef[];
  readonly names: Names;
  readonly picked: boolean;
  readonly onPick: (id: string, range: boolean) => void;
};

/** One Work's row. It draws again only when its row, its place or its pick changes. */
export const LibraryRow = memo(function LibraryRow({
  index,
  start,
  measure,
  columns,
  names,
  picked,
  onPick,
  ...cell
}: RowProps) {
  const { work } = cell;
  return (
    <tr
      ref={measure}
      data-index={index}
      data-work-id={work.id}
      data-library-row=""
      aria-rowindex={index + 2}
      className={`${rowLook} ${work.monitored === "none" ? "text-ink-2" : ""}`}
      style={{ transform: `translateY(${start}px)` }}
    >
      <td data-col="pick">
        <Checkbox
          label={`Pick ${work.label}`}
          hideLabel
          checked={picked}
          onCheckedChange={(_, by) => {
            onPick(work.id, by.shiftKey);
          }}
        />
      </td>
      <td data-col="work">
        <TitleCell {...cell} />
      </td>
      {columns.map((column) => (
        <td
          key={column.key}
          data-col={column.key}
          data-label={column.header}
          data-align={column.align}
        >
          {column.cell(work, names)}
        </td>
      ))}
    </tr>
  );
});

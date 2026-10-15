import { type ReactNode, useId } from "react";

import { moveBetweenRows, rowGroup, rowTarget } from "./row-keys";
import "./rows.css";

export type Column<Row> = {
  id: string;
  header: string;
  cell: (row: Row) => ReactNode;
  hideBelow?: number;
  align?: "start" | "end";
};

export type Selection<Row> = {
  selected: string | undefined;
  onSelect: (row: Row) => void;
};

const lineHeight = { 1: "h-(--row-height)", 2: "h-(--row-height-two)" } as const;

const alignLook = { start: "text-left", end: "text-right" } as const;

function reflow(
  table: string,
  columns: readonly { id: string; header: string; hideBelow?: number }[],
  stackBelow: number | undefined,
) {
  const scope = `[data-table="${table}"]`;
  const hidden = columns.flatMap(({ id, hideBelow }) =>
    hideBelow === undefined
      ? []
      : [`@container (width < ${hideBelow}px) { ${scope} [data-col="${id}"] { display: none; } }`],
  );
  const stacked =
    stackBelow === undefined
      ? []
      : [
          `@container (width < ${stackBelow}px) {
  ${scope} thead { display: none; }
  ${scope} :is(table, tbody, tr) { display: block; }
  ${scope} tr { padding: 12px 16px; }
  ${scope} tr + tr { box-shadow: inset 0 1px 0 var(--color-hairline); }
  ${scope} td { display: block; height: auto; padding: 2px 0; box-shadow: none; text-align: start; }
  ${scope} td::before { content: attr(data-label); display: block; font-size: var(--text-label); color: var(--color-ink-3); }
}`,
        ];
  return [...hidden, ...stacked].join("\n");
}

function Cell<Row>({
  column,
  row,
  lines,
  target,
}: {
  column: Column<Row>;
  row: Row;
  lines: 1 | 2;
  target: ReactNode;
}) {
  return (
    <td
      data-col={column.id}
      data-label={column.header}
      className={`px-4 align-middle text-(length:--body-size) leading-(--body-line-height) text-ink-1 ${lineHeight[lines]} ${alignLook[column.align ?? "start"]}`}
    >
      {target ?? column.cell(row)}
    </td>
  );
}

function Head({
  columns,
}: {
  columns: readonly { id: string; header: string; align?: "start" | "end" }[];
}) {
  return (
    <thead>
      <tr>
        {columns.map(({ id: col, header, align }) => (
          <th
            key={col}
            data-col={col}
            scope="col"
            className={`h-8 px-4 text-label text-ink-3 ${alignLook[align ?? "start"]}`}
          >
            {header}
          </th>
        ))}
      </tr>
    </thead>
  );
}

function Row<Row>({
  row,
  columns: [first, ...rest],
  lines,
  selection,
}: {
  row: Row;
  columns: readonly [Column<Row>, ...Column<Row>[]];
  lines: 1 | 2;
  selection: { selected: boolean; onSelect: (row: Row) => void } | undefined;
}) {
  return (
    <tr className="relative transition-colors duration-350 ease-standard *:shadow-[inset_0_1px_0_var(--color-hairline)] hover:bg-(--row-wash) has-focus-visible:bg-(--row-wash) has-focus-visible:outline-2 has-focus-visible:-outline-offset-2 has-focus-visible:outline-solid">
      <Cell
        column={first}
        row={row}
        lines={lines}
        target={
          selection && (
            <button
              type="button"
              {...{ [rowTarget]: "" }}
              aria-pressed={selection.selected}
              onClick={() => {
                selection.onSelect(row);
              }}
              onKeyDown={(event) => {
                moveBetweenRows(event)?.click();
              }}
              className="text-left after:absolute after:inset-0 focus-visible:shadow-none focus-visible:outline-none"
            >
              {selection.selected && (
                <span
                  aria-hidden="true"
                  className="absolute inset-y-2 left-0 w-[3px] rounded-capsule bg-solid shadow-live-glow"
                />
              )}
              {first.cell(row)}
            </button>
          )
        }
      />
      {rest.map((column) => (
        <Cell key={column.id} column={column} row={row} lines={lines} target={undefined} />
      ))}
    </tr>
  );
}

export function Table<Row>({
  label,
  columns,
  rows,
  rowKey,
  lines = 1,
  stackBelow,
  selection,
}: {
  label: string;
  columns: readonly [Column<Row>, ...Column<Row>[]];
  rows: readonly Row[];
  rowKey: (row: Row) => string;
  lines?: 1 | 2;
  stackBelow?: number;
  selection?: Selection<Row>;
}) {
  const id = useId();
  return (
    <div className="@container">
      <style>{reflow(id, columns, stackBelow)}</style>
      <div data-table={id} className="rounded-card bg-glass ring-1 ring-hairline ring-inset">
        <table aria-label={label} className="w-full border-separate border-spacing-0">
          <Head columns={columns} />
          <tbody {...{ [rowGroup]: "" }}>
            {rows.map((row) => (
              <Row
                key={rowKey(row)}
                row={row}
                columns={columns}
                lines={lines}
                selection={
                  selection && { ...selection, selected: selection.selected === rowKey(row) }
                }
              />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

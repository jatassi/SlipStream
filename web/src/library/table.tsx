import { useWindowVirtualizer } from "@tanstack/react-virtual";
import { useCallback, type ReactNode } from "react";

import { useShell } from "../ui/shell-context";
import type { Arranged } from "./arrange";
import type { ColumnDef, Names } from "./columns";
import { tableMove } from "./keyboard";
import { type Picking, useBrowsing } from "./picking";
import { TableHead } from "./table-head";
import { LibraryRow } from "./table-row";
import { useScrollMargin } from "./window-list";
import "./table.css";

declare module "react" {
  interface CSSProperties {
    /** The tracks of the columns a Library table shows, after the pick and the title. */
    "--library-tracks"?: string;
  }
}

const rowEstimate = { desktop: 60, phone: 64 } as const;
const stackedLine = 40;

type TableProps = {
  readonly arranged: Arranged;
  readonly columns: readonly ColumnDef[];
  readonly names: Names;
  readonly picking: Picking;
  readonly label: string;
};

function TableFrame({
  arranged,
  columns,
  picking,
  label,
  children,
}: Omit<TableProps, "names"> & { children: ReactNode }) {
  return (
    <div
      className="library-table"
      style={{ "--library-tracks": columns.map(({ track }) => track).join(" ") }}
    >
      <table
        aria-label={label}
        aria-rowcount={arranged.works.length + 1}
        className="block rounded-card bg-glass ring-1 ring-hairline ring-inset"
      >
        <TableHead columns={columns} picking={picking} shown={arranged.order.length} />
        {children}
      </table>
    </div>
  );
}

/**
 * The virtualizer's callbacks, which change only with the order or the
 * shell: the virtualizer lays out every row again whenever getItemKey
 * changes.
 */
function useRowSizing(order: Arranged["order"], columnCount: number) {
  const shell = useShell();
  const rowHeight = rowEstimate[shell] + (shell === "phone" ? stackedLine * columnCount : 0);
  const estimateSize = useCallback(() => rowHeight, [rowHeight]);
  const getItemKey = useCallback((index: number) => order[index] ?? index, [order]);
  return { estimateSize, getItemKey };
}

/**
 * One row per Work with the columns the view picks, drawn only near the
 * window's view. Below 620px of its own width each row stacks its values
 * under their labels.
 */
export function LibraryTable({ arranged, columns, names, picking, label }: TableProps) {
  const body = useScrollMargin<HTMLTableSectionElement>();
  const { estimateSize, getItemKey } = useRowSizing(arranged.order, columns.length);
  // oxlint-disable-next-line react/incompatible-library -- this leaf holds the virtualizer so the compiler's bail-out stays here
  const virtualizer = useWindowVirtualizer({
    count: arranged.works.length,
    estimateSize,
    overscan: 8,
    scrollMargin: body.margin,
    getItemKey,
  });
  const { tabbable, handlers } = useBrowsing({
    order: arranged.order,
    picking,
    move: tableMove,
    reveal: virtualizer.scrollToIndex,
  });
  return (
    <TableFrame arranged={arranged} columns={columns} picking={picking} label={label}>
      <tbody
        ref={body.ref}
        className="relative block"
        style={{ height: virtualizer.getTotalSize() }}
      >
        {virtualizer.getVirtualItems().map((item) => {
          const work = arranged.works[item.index];
          return (
            work && (
              <LibraryRow
                key={item.key}
                work={work}
                index={item.index}
                start={item.start - virtualizer.options.scrollMargin}
                measure={virtualizer.measureElement}
                picked={picking.has(work.id)}
                tabbable={tabbable === work.id}
                onPick={picking.toggle}
                {...{ columns, names, handlers }}
              />
            )
          );
        })}
      </tbody>
    </TableFrame>
  );
}

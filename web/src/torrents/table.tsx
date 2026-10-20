import { useWindowVirtualizer } from "@tanstack/react-virtual";
import { type ComponentType, type ReactNode, useLayoutEffect, useRef, useState } from "react";

import type { Clock } from "../format";
import { useLiveRow } from "../live/torrents";
import type { Selection } from "../ui/table";
import type { Row } from "./model";
import type { Names } from "./names";
import {
  type CellProps,
  NodeCell,
  ProgressCell,
  RatioCell,
  SpeedCell,
  StateCell,
  TrackerCell,
  WorkCell,
} from "./row";
import "../ui/rows.css";
import "./table.css";

type Column = {
  readonly id: string;
  readonly header: string;
  readonly cell: ComponentType<CellProps>;
};

const columns: readonly Column[] = [
  { id: "work", header: "Work", cell: WorkCell },
  { id: "state", header: "State", cell: StateCell },
  { id: "progress", header: "Progress", cell: ProgressCell },
  { id: "ratio", header: "Ratio", cell: RatioCell },
  { id: "speed", header: "Speed", cell: SpeedCell },
  { id: "tracker", header: "Tracker", cell: TrackerCell },
  { id: "node", header: "Node · client", cell: NodeCell },
];

const rowLook =
  "absolute inset-x-0 top-0 min-h-(--row-height-two) text-(length:--body-size) leading-(--body-line-height) text-ink-1 shadow-[inset_0_1px_0_var(--color-hairline)] transition-colors duration-350 ease-standard hover:bg-(--row-wash) has-focus-visible:bg-(--row-wash) has-focus-visible:outline-2 has-focus-visible:-outline-offset-2 has-focus-visible:outline-solid";

/** Where the Inspector and the row menu plug in: the selected row and each row's menu. */
export type RowSeams = {
  readonly selection?: Selection<Row> | undefined;
  readonly menu?: ((row: Row) => ReactNode) | undefined;
};

type Shared = RowSeams & { readonly names: Names; readonly clock: Clock };

function WorkTarget({ props, selection }: { props: CellProps; selection: Selection<Row> }) {
  const selected = selection.selected === props.row.id;
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={() => {
        selection.onSelect(props.row);
      }}
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

function TorrentRow({
  row,
  index,
  start,
  measure,
  names,
  clock,
  selection,
  menu,
}: Shared & {
  row: Row;
  index: number;
  start: number;
  measure: (element: HTMLElement | null) => void;
}) {
  const props: CellProps = { row, live: useLiveRow(row.id), names, clock };
  return (
    <tr
      ref={measure}
      data-index={index}
      aria-rowindex={index + 2}
      data-torrent-row=""
      className={rowLook}
      style={{ transform: `translateY(${start}px)` }}
    >
      {columns.map(({ id, header, cell: Cell }) => (
        <td key={id} data-col={id} data-label={header} className="min-w-0">
          {id === "work" && selection !== undefined ? (
            <WorkTarget props={props} selection={selection} />
          ) : (
            <Cell {...props} />
          )}
        </td>
      ))}
      {menu && <td data-col="menu">{menu(row)}</td>}
    </tr>
  );
}

function Head({ menu }: { menu: boolean }) {
  return (
    <thead data-torrent-head="" className="block">
      <tr aria-rowindex={1} data-torrent-row="" className="h-8 text-label text-ink-3">
        {columns.map(({ id, header }) => (
          <th key={id} scope="col" data-col={id} className="truncate text-left font-medium">
            {header}
          </th>
        ))}
        {menu && (
          <th scope="col" data-col="menu">
            <span className="sr-only">Actions</span>
          </th>
        )}
      </tr>
    </thead>
  );
}

// Where the body starts on the page, which the window's scroll is measured from.
function useOffset<E extends HTMLElement>() {
  const ref = useRef<E>(null);
  const [offset, setOffset] = useState(0);
  useLayoutEffect(() => {
    const measure = () => {
      setOffset((ref.current?.getBoundingClientRect().top ?? 0) + window.scrollY);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(document.body);
    return () => {
      observer.disconnect();
    };
  }, []);
  return { ref, offset };
}

/**
 * Every Torrent the filters leave, drawn only near the window's view, so a
 * seedbox of thousands scrolls smoothly. Columns drop at 980px and rows
 * become cards at 620px of its own width.
 */
export function TorrentTable({
  rows,
  estimate,
  ...shared
}: Shared & { rows: readonly Row[]; estimate: number }) {
  const body = useOffset<HTMLTableSectionElement>();
  // oxlint-disable-next-line react/incompatible-library -- this leaf holds the virtualizer so the compiler's bail-out stays here
  const virtualizer = useWindowVirtualizer({
    count: rows.length,
    estimateSize: () => estimate,
    overscan: 8,
    scrollMargin: body.offset,
    getItemKey: (index) => rows[index]?.id ?? index,
  });
  return (
    <div className="torrents">
      <table
        aria-label="Torrents"
        aria-rowcount={rows.length + 1}
        className="block rounded-card bg-glass ring-1 ring-hairline ring-inset"
      >
        <Head menu={shared.menu !== undefined} />
        <tbody
          ref={body.ref}
          className="relative block"
          style={{ height: virtualizer.getTotalSize() }}
        >
          {virtualizer.getVirtualItems().map((item) => {
            const row = rows[item.index];
            return (
              row && (
                <TorrentRow
                  key={item.key}
                  row={row}
                  index={item.index}
                  start={item.start - virtualizer.options.scrollMargin}
                  measure={virtualizer.measureElement}
                  {...shared}
                />
              )
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

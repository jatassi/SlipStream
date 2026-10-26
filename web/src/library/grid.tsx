import { useWindowVirtualizer } from "@tanstack/react-virtual";
import { useCallback, useMemo } from "react";

import type { Arranged } from "./arrange";
import { type GridRow, gridRows } from "./grid-layout";
import { gridMove } from "./keyboard";
import { type CellHandlers, type Picking, useBrowsing } from "./picking";
import { PosterCard } from "./poster-card";
import type { PosterSize } from "./view";
import { useScrollMargin, useWidth } from "./window-list";

const narrowest: { readonly [S in PosterSize]: number } = { small: 120, medium: 160, large: 240 };

const gap = 16;
// The label, the fact and the status under each poster.
const linesUnder = 72;
const headingHeight = 52;

function heightOf(rows: readonly GridRow[], width: number, perRow: number) {
  const poster = ((width - gap * (perRow - 1)) / perRow) * 1.5;
  return (index: number) =>
    rows[index]?.kind === "section" ? headingHeight : poster + linesUnder + gap;
}

const rowKey = (row: GridRow | undefined, index: number) =>
  row === undefined ? index : row.kind === "section" ? `s:${row.section.key}` : `p:${row.first}`;

type GridProps = {
  readonly arranged: Arranged;
  readonly size: PosterSize;
  readonly picking: Picking;
};

function Row({
  row,
  perRow,
  arranged,
  picking,
  tabbable,
  handlers,
}: {
  row: GridRow;
  perRow: number;
  arranged: Arranged;
  picking: Picking;
  tabbable: string | undefined;
  handlers: CellHandlers;
}) {
  if (row.kind === "section") {
    return (
      <h2 className="flex items-baseline gap-2 pt-4 pb-3 text-heading text-ink-1">
        {row.section.label} <span className="text-label text-ink-3 tabular-nums">{row.count}</span>
      </h2>
    );
  }
  return (
    <ul
      className="grid pb-4"
      style={{ gap, gridTemplateColumns: `repeat(${perRow}, minmax(0, 1fr))` }}
    >
      {row.ids.map((id, column) => {
        const work = arranged.works[row.first + column];
        return (
          work && (
            <PosterCard
              key={id}
              work={work}
              picked={picking.has(id)}
              selecting={picking.selecting}
              tabbable={tabbable === id}
              handlers={handlers}
              onPick={picking.toggle}
            />
          )
        );
      })}
    </ul>
  );
}

/**
 * The grid's rows and the virtualizer's callbacks, which change only with
 * the Works or the width: the virtualizer lays out every row again whenever
 * getItemKey changes.
 */
function useLayout(arranged: Arranged, perRow: number, width: number) {
  const layout = useMemo(() => gridRows(arranged, perRow), [arranged, perRow]);
  const estimateSize = useMemo(() => heightOf(layout.rows, width, perRow), [layout, width, perRow]);
  const getItemKey = useCallback((index: number) => rowKey(layout.rows[index], index), [layout]);
  return { layout, estimateSize, getItemKey };
}

/**
 * The poster grid, drawn only near the window's view so 20,000 Works
 * scroll smoothly, with a heading over each section. Posters load as
 * their row is drawn.
 */
export function LibraryGrid({ arranged, size, picking }: GridProps) {
  const sized = useWidth<HTMLDivElement>();
  const perRow = Math.max(1, Math.floor((sized.width + gap) / (narrowest[size] + gap)));
  const { layout, estimateSize, getItemKey } = useLayout(arranged, perRow, sized.width);
  const body = useScrollMargin<HTMLDivElement>();
  // oxlint-disable-next-line react/incompatible-library -- this leaf holds the virtualizer so the compiler's bail-out stays here
  const virtualizer = useWindowVirtualizer({
    count: layout.rows.length,
    estimateSize,
    overscan: 3,
    scrollMargin: body.margin,
    getItemKey,
  });
  const { tabbable, handlers } = useBrowsing({
    order: arranged.order,
    picking,
    move: (index, step, count) => gridMove(layout, index, step, count),
    reveal: (index) => {
      virtualizer.scrollToIndex(layout.rowOf(index));
    },
  });
  return (
    <div ref={sized.ref}>
      <div ref={body.ref} className="relative" style={{ height: virtualizer.getTotalSize() }}>
        {virtualizer.getVirtualItems().map((item) => {
          const row = layout.rows[item.index];
          return (
            row && (
              <div
                key={item.key}
                ref={virtualizer.measureElement}
                data-index={item.index}
                className="absolute inset-x-0 top-0"
                style={{
                  transform: `translateY(${item.start - virtualizer.options.scrollMargin}px)`,
                }}
              >
                <Row row={row} perRow={perRow} {...{ arranged, picking, tabbable, handlers }} />
              </div>
            )
          );
        })}
      </div>
    </div>
  );
}

import { useVirtualizer } from "@tanstack/react-virtual";
import { type ReactNode, useEffect, useRef } from "react";

// Within this many pixels of the end, the list counts as at its tail.
const tailSlack = 4;

function rowAt<R>(rows: readonly R[], index: number): R {
  const row = rows[index];
  if (row === undefined) {
    throw new RangeError(`no row at ${index} of ${rows.length}`);
  }
  return row;
}

const atTail = ({ scrollHeight, scrollTop, clientHeight }: HTMLElement) =>
  scrollHeight - scrollTop - clientHeight <= tailSlack;

function followTail(
  virtualizer: { scrollToIndex: (index: number, options: { align: "end" }) => void },
  count: number,
  follow: boolean,
) {
  if (follow && count > 0) {
    virtualizer.scrollToIndex(count - 1, { align: "end" });
  }
}

export type VirtualRowsProps<R> = {
  label: string;
  rows: readonly R[];
  rowKey: (row: R) => string | number;
  estimate: number;
  follow: boolean;
  onFollowChange: (follow: boolean) => void;
  renderRow: (row: R) => ReactNode;
};

/**
 * A list too long to draw whole: only the rows in view are in the DOM. It
 * follows the tail while `follow` is on, and reports scrolling away from the
 * tail or back to it through `onFollowChange`.
 */
export function VirtualRows<R>({
  label,
  rows,
  rowKey,
  estimate,
  follow,
  onFollowChange,
  renderRow,
}: VirtualRowsProps<R>) {
  const scroller = useRef<HTMLDivElement>(null);
  // oxlint-disable-next-line react/incompatible-library -- this leaf holds the virtualizer so the compiler's bail-out stays here
  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scroller.current,
    estimateSize: () => estimate,
    getItemKey: (index) => rowKey(rowAt(rows, index)),
    overscan: 10,
  });

  useEffect(() => {
    followTail(virtualizer, rows.length, follow);
  }, [follow, rows, virtualizer]);

  return (
    <div
      ref={scroller}
      onScroll={(event) => {
        const tail = atTail(event.currentTarget);
        if (tail !== follow) {
          onFollowChange(tail);
        }
      }}
      className="min-h-0 flex-1 overflow-y-auto overscroll-contain"
    >
      <ul aria-label={label} className="relative" style={{ height: virtualizer.getTotalSize() }}>
        {virtualizer.getVirtualItems().map((item) => (
          <li
            key={item.key}
            ref={virtualizer.measureElement}
            data-index={item.index}
            className="absolute inset-x-0 top-0"
            style={{ transform: `translateY(${item.start}px)` }}
          >
            {renderRow(rowAt(rows, item.index))}
          </li>
        ))}
      </ul>
    </div>
  );
}

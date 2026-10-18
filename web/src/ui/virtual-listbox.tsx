import { useVirtualizer } from "@tanstack/react-virtual";
import { type ReactNode, useEffect, useRef } from "react";

function rowAt<R>(rows: readonly R[], index: number): R {
  const row = rows[index];
  if (row === undefined) {
    throw new RangeError(`no row at ${index} of ${rows.length}`);
  }
  return row;
}

function useKeptInView(
  virtualizer: { scrollToIndex: (index: number) => void },
  active: number,
  rows: readonly unknown[],
) {
  useEffect(() => {
    if (active >= 0 && active < rows.length) {
      virtualizer.scrollToIndex(active);
    }
  }, [active, rows, virtualizer]);
}

/** The id of a listbox's option, for the owning field's aria-activedescendant. */
export const optionId = (listbox: string, index: number) => `${listbox}-${index}`;

export type VirtualListboxProps<R> = {
  readonly id: string;
  readonly label: string;
  readonly rows: readonly R[];
  readonly rowKey: (row: R) => string;
  readonly estimate: number;
  /** The active row, kept in view; anything outside the rows is none. */
  readonly active: number;
  readonly onChoose: (index: number) => void;
  readonly renderRow: (row: R, active: boolean) => ReactNode;
};

/**
 * VirtualRows' sibling for a list a field drives: focus stays in the field,
 * which names the active option through aria-activedescendant and moves it by
 * index. Only the rows in view are in the DOM, so each sets its place in the
 * whole. No tail following.
 */
export function VirtualListbox<R>({
  id,
  label,
  rows,
  rowKey,
  estimate,
  active,
  onChoose,
  renderRow,
}: VirtualListboxProps<R>) {
  const scroller = useRef<HTMLDivElement>(null);
  // oxlint-disable-next-line react/incompatible-library -- this leaf holds the virtualizer so the compiler's bail-out stays here
  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scroller.current,
    estimateSize: () => estimate,
    getItemKey: (index) => rowKey(rowAt(rows, index)),
    overscan: 10,
  });

  useKeptInView(virtualizer, active, rows);

  return (
    <div ref={scroller} className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
      <ul
        id={id}
        // oxlint-disable-next-line jsx-a11y/prefer-tag-over-role, jsx-a11y/no-noninteractive-element-to-interactive-role -- a select can't be virtualized or driven by aria-activedescendant
        role="listbox"
        aria-label={label}
        className="relative"
        style={{ height: virtualizer.getTotalSize() }}
      >
        {virtualizer.getVirtualItems().map((item) => (
          <Option
            key={item.key}
            id={optionId(id, item.index)}
            index={item.index}
            start={item.start}
            count={rows.length}
            active={item.index === active}
            measure={virtualizer.measureElement}
            onChoose={onChoose}
          >
            {renderRow(rowAt(rows, item.index), item.index === active)}
          </Option>
        ))}
      </ul>
    </div>
  );
}

function Option({
  id,
  index,
  start,
  count,
  active,
  measure,
  onChoose,
  children,
}: {
  readonly id: string;
  readonly index: number;
  readonly start: number;
  readonly count: number;
  readonly active: boolean;
  readonly measure: (element: HTMLLIElement | null) => void;
  readonly onChoose: (index: number) => void;
  readonly children: ReactNode;
}) {
  return (
    // oxlint-disable-next-line jsx-a11y/click-events-have-key-events -- keys reach the option through the field that names it in aria-activedescendant
    <li
      ref={measure}
      id={id}
      // oxlint-disable-next-line jsx-a11y/prefer-tag-over-role, jsx-a11y/no-noninteractive-element-to-interactive-role -- the option of a virtualized listbox
      role="option"
      aria-selected={active}
      aria-setsize={count}
      aria-posinset={index + 1}
      data-index={index}
      className="absolute inset-x-0 top-0"
      style={{ transform: `translateY(${start}px)` }}
      onMouseDown={(event) => {
        // The field keeps focus, so the keys keep working after a click.
        event.preventDefault();
      }}
      onClick={() => {
        onChoose(index);
      }}
    >
      {children}
    </li>
  );
}

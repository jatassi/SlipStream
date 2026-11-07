import { useWindowVirtualizer } from "@tanstack/react-virtual";
import { useCallback } from "react";

import { useScrollMargin } from "../../library/window-list";
import type { CellPopoverControls } from "../../ui/cell-popover";
import type { PoppedCell } from "./cell-detail";
import { FoldRow, ItemRow, type RowActions } from "./item-row";
import { FooterRow, type HeaderActions, LevelHeader } from "./level-header";
import type { LevelContext } from "./model";
import { PhoneItemRow } from "./phone-row";
import type { Row } from "./rows";

/** What every drawn row shares, each the same object across a page's renders. */
export type Drawn = {
  readonly phone: boolean;
  readonly context: LevelContext;
  readonly controls: CellPopoverControls<PoppedCell>;
  readonly rowActions: RowActions;
  readonly headerActions: HeaderActions;
};

function sizeOf(row: Row | undefined, index: number, phone: boolean): number {
  switch (row?.kind) {
    case "header":
      return index === 0 ? 56 : 68;
    case "footer":
      return 12;
    case "item":
    case "fold":
    case "withdrawn":
      return phone ? 52 : 36;
    case undefined:
      return 0;
    default:
      return row satisfies never;
  }
}

function RowView({ row, first, drawn }: { row: Row; first: boolean; drawn: Drawn }) {
  const { phone, context, controls, rowActions, headerActions } = drawn;
  const source = context.type.source;
  switch (row.kind) {
    case "header":
      return (
        <LevelHeader
          level={row.level}
          open={row.open}
          first={first}
          phone={phone}
          targets={context.targets}
          source={source}
          actions={headerActions}
        />
      );
    case "item":
    case "withdrawn":
      return phone ? (
        <PhoneItemRow view={row.item} source={source} actions={rowActions} />
      ) : (
        <ItemRow
          view={row.item}
          columns={row.level.columns}
          targets={context.targets}
          source={source}
          controls={controls}
          actions={rowActions}
        />
      );
    case "fold":
      return <FoldRow fold={row.fold} phone={phone} />;
    case "footer":
      return <FooterRow />;
    default:
      return row satisfies never;
  }
}

/** The rows near the window's view, so a Work of 1,200 Items draws about a screenful. */
export function VirtualLevels({ rows, drawn }: { rows: readonly Row[]; drawn: Drawn }) {
  const body = useScrollMargin<HTMLDivElement>();
  const { phone } = drawn;
  const estimateSize = useCallback(
    (index: number) => sizeOf(rows[index], index, phone),
    [rows, phone],
  );
  const getItemKey = useCallback((index: number) => rows[index]?.key ?? index, [rows]);
  // oxlint-disable-next-line react/incompatible-library -- this leaf holds the virtualizer so the compiler's bail-out stays here
  const virtualizer = useWindowVirtualizer({
    count: rows.length,
    estimateSize,
    overscan: 6,
    scrollMargin: body.margin,
    getItemKey,
  });
  return (
    <div ref={body.ref} className="relative" style={{ height: virtualizer.getTotalSize() }}>
      {virtualizer.getVirtualItems().map((item) => {
        const row = rows[item.index];
        return (
          row && (
            <div
              key={item.key}
              data-index={item.index}
              className="absolute inset-x-0 top-0"
              style={{
                height: item.size,
                transform: `translateY(${item.start - virtualizer.options.scrollMargin}px)`,
              }}
            >
              <RowView row={row} first={item.index === 0} drawn={drawn} />
            </div>
          )
        );
      })}
    </div>
  );
}

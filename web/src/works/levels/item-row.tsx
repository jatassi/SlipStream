import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { memo } from "react";

import type { CellPopoverControls } from "../../ui/cell-popover";
import { IconButton } from "../../ui/icon-button";
import { RowSwitch } from "../../ui/row-switch";
import { dayWords } from "../page/fields-format";
import type { PoppedCell } from "./cell-detail";
import { Squares } from "./columns";
import type { Fold } from "./fold";
import type { Column, ItemView, LevelTarget } from "./model";

/** What a row can do, the same object for every row. */
export type RowActions = {
  readonly monitor: (view: ItemView, on: boolean) => void;
  readonly details: (view: ItemView) => void;
  /** Phone: open the row's sheet. */
  readonly open: (view: ItemView) => void;
};

export const cardBody = "bg-glass px-4 shadow-[inset_0_1px_0_var(--color-hairline)]";

/** "Not monitored", "No longer listed by TVDB", or nothing for a monitored live Item. */
export const quietWords = (view: ItemView, source: string) =>
  view.withdrawn ? `No longer listed by ${source}` : view.quiet ? "Not monitored" : undefined;

/** The label and what quiets it, then the availability date. */
export function Says({ view, source }: { view: ItemView; source: string }) {
  const quiet = quietWords(view, source);
  return (
    <span className="flex min-w-0 flex-1 items-baseline gap-2">
      <span className={`truncate ${quiet === undefined ? "text-ink-1" : "text-ink-2"}`}>
        {view.item.label}
      </span>
      {quiet !== undefined && <span className="shrink-0 text-meta text-ink-3">{quiet}</span>}
    </span>
  );
}

/** A 36px Item row: code, label, date, squares, then its Monitor switch, a Search slot and Details. */
export const ItemRow = memo(function ItemRow({
  view,
  columns,
  targets,
  source,
  controls,
  actions,
}: {
  view: ItemView;
  columns: readonly Column[];
  targets: readonly LevelTarget[];
  source: string;
  controls: CellPopoverControls<PoppedCell>;
  actions: RowActions;
}) {
  return (
    <div className={`flex h-9 items-center gap-3 ${cardBody}`}>
      <span className="w-20 shrink-0 truncate text-meta text-ink-2 tabular-nums">{view.code}</span>
      <Says view={view} source={source} />
      <span className="w-40 shrink-0 truncate text-meta text-ink-3 tabular-nums">
        {view.date === null ? "" : dayWords(view.date)}
      </span>
      <Squares view={view} columns={columns} targets={targets} controls={controls} />
      <span className="flex w-8.5 shrink-0">
        {!view.withdrawn && (
          <RowSwitch
            label={`Monitor ${view.code}`}
            checked={view.item.monitored}
            onCheckedChange={(on) => {
              actions.monitor(view, on);
            }}
          />
        )}
      </span>
      <span aria-hidden="true" className="w-8 shrink-0" />
      <IconButton
        icon={MoreHorizontalIcon}
        label={`Details of ${view.code}`}
        onClick={() => {
          actions.details(view);
        }}
      />
    </div>
  );
});

/** A run of unreleased Items as one line. */
export function FoldRow({ fold, phone }: { fold: Fold<ItemView>; phone: boolean }) {
  return (
    <div className={`flex items-center gap-3 ${phone ? "h-13" : "h-9"} ${cardBody}`}>
      <span className={`${phone ? "w-16" : "w-20"} shrink-0`} />
      <span className="truncate text-ink-3">{fold.words}</span>
    </div>
  );
}

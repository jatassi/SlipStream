import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { memo } from "react";

import { Icon } from "../../ui/icon";
import { StateSquare } from "../../ui/state-square";
import { type RowActions, Says, cardBody } from "./item-row";
import type { ItemView } from "./model";

/** Phone: the row is one target that opens the Item's sheet. */
export const PhoneItemRow = memo(function PhoneItemRow({
  view,
  source,
  actions,
}: {
  view: ItemView;
  source: string;
  actions: RowActions;
}) {
  return (
    <div className={`flex h-13 items-center ${cardBody}`}>
      <button
        type="button"
        onClick={() => {
          actions.open(view);
        }}
        className="flex h-full min-w-0 flex-1 items-center gap-3 text-left"
      >
        <span className="w-16 shrink-0 truncate text-meta text-ink-2 tabular-nums">
          {view.code}
        </span>
        <Says view={view} source={source} />
        {view.status !== null && <StateSquare state={view.status} quiet={view.quiet} />}
        <Icon icon={ArrowRight01Icon} className="size-4 shrink-0 text-ink-3" />
      </button>
    </div>
  );
});

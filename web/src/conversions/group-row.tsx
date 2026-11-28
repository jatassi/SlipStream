import { useState } from "react";

import type { ConversionsConversionGroup } from "../api/gen/model";
import { bytes, spanWords } from "../format";
import { Chevron } from "../ui/chevron";
import { CancelGroupDialog, countWords } from "./cancel-dialogs";
import { GroupRows } from "./group-rows";
import { Lines, RowMenu, listed, rowLook } from "./row-parts";

const groupWords = (group: ConversionsConversionGroup) =>
  [
    countWords(group.count),
    group.nodes.length > 0 && `on ${listed.format(group.nodes)}`,
    `about ${spanWords(group.seconds)}`,
    bytes(group.bytes),
  ]
    .filter((part) => part !== false)
    .join(" · ");

/** A backfill or a season folded into one row, which opens its Conversions in place. */
export function GroupRow({ group }: { group: ConversionsConversionGroup }) {
  const [open, setOpen] = useState(false);
  const [canceling, setCanceling] = useState(false);
  return (
    <div className="flex flex-col gap-1.5">
      <div className={rowLook}>
        <button
          type="button"
          aria-expanded={open}
          data-panel-open={open || undefined}
          className="group flex min-w-0 flex-1 items-center gap-3 text-left"
          onClick={() => {
            setOpen(!open);
          }}
        >
          <span className="flex text-ink-4">
            <Chevron />
          </span>
          <Lines first={group.label} second={groupWords(group)} />
        </button>
        <RowMenu
          label={group.label}
          items={[
            {
              label: "Cancel all…",
              tone: "danger",
              onSelect: () => {
                setCanceling(true);
              },
            },
          ]}
        />
      </div>
      {open && <GroupRows group={group} />}
      <CancelGroupDialog group={group} open={canceling} onOpenChange={setCanceling} />
    </div>
  );
}

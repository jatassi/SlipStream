import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { HealthCategoryID, HealthItem } from "../api/gen/model";
import { Data } from "../ui/data";
import { Icon } from "../ui/icon";
import { Sheet } from "../ui/sheet";
import { SignalChip } from "../ui/signal-chip";
import { signalLook } from "../ui/signal-row";
import { ItemBody } from "./item-body";
import { levels } from "./levels";

export function PhoneItem({
  category,
  item,
  since,
}: {
  category: HealthCategoryID;
  item: HealthItem;
  since: string;
}) {
  const [open, setOpen] = useState(false);
  const look = signalLook[item.level];
  const chip = <SignalChip kind={item.level}>{levels[item.level].chip}</SignalChip>;
  return (
    <Sheet
      title={item.subject}
      open={open}
      onOpenChange={setOpen}
      trigger={
        <button
          type="button"
          className={`flex min-h-(--control-height) w-full items-center gap-3 rounded-card px-4 py-3 text-left text-(length:--body-size) leading-(--body-line-height) ${look.row}`}
        >
          {chip}
          <span className="min-w-0 flex-1">
            <span className="block truncate font-semibold">{item.subject}</span>
            <span className={`line-clamp-2 ${look.dim}`}>{item.text}</span>
          </span>
          <Icon icon={ArrowRight01Icon} className={`size-4 shrink-0 ${look.dim}`} />
        </button>
      }
    >
      <div className={`flex flex-col items-start gap-3 rounded-card p-4 ${look.row}`}>
        {chip}
        <p className={look.dim}>{item.text}</p>
        <p className={look.dim}>
          since <Data>{since}</Data>
        </p>
        <ItemBody category={category} item={item} />
      </div>
    </Sheet>
  );
}

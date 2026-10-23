import type { HealthCategoryID, HealthItem } from "../api/gen/model";
import { type Clock, sinceWords } from "../format";
import { useShell } from "../ui/shell-context";
import { SignalRow } from "../ui/signal-row";
import { ItemBody } from "./item-body";
import { itemAnchor, levels } from "./levels";
import { PhoneItem } from "./phone-item";

export type ItemProps = {
  category: HealthCategoryID;
  item: HealthItem;
  reading: Clock;
};

export function ItemRow({ category, item, reading }: ItemProps) {
  const since = sinceWords(item.since, reading);
  return (
    <div id={itemAnchor(category, item.key)} className="scroll-mt-20">
      {useShell() === "phone" ? (
        <PhoneItem category={category} item={item} since={since} />
      ) : (
        <SignalRow
          signal={item.level}
          chip={levels[item.level].chip}
          subject={item.subject}
          verdict={item.text}
          timeLead="since"
          time={since}
        >
          <ItemBody category={category} item={item} />
        </SignalRow>
      )}
    </div>
  );
}

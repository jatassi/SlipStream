import type { HealthCategoryID, HealthFact, HealthItem } from "../api/gen/model";
import { bytes, clock } from "../format";
import { Data } from "../ui/data";
import { Fact, Facts } from "../ui/facts";
import { signalLook } from "../ui/signal-row";
import { ItemAction } from "./item-action";

export function ItemBody({ category, item }: { category: HealthCategoryID; item: HealthItem }) {
  const { dim } = signalLook[item.level];
  return (
    <div className="flex w-full flex-col items-start gap-3">
      {item.unsure && (
        <p className={dim}>
          SlipStream couldn&apos;t check this item. It shows what the last successful check found.
        </p>
      )}
      {item.detail.length > 0 && (
        <div className="w-full">
          <Facts look="well" signal={item.level}>
            {item.detail.map((fact) => (
              <Fact key={fact.label} label={fact.label}>
                <FactValue fact={fact} />
              </Fact>
            ))}
          </Facts>
        </div>
      )}
      {item.action !== null && <ItemAction category={category} item={item} action={item.action} />}
    </div>
  );
}

function FactValue({ fact }: { fact: HealthFact }) {
  if (fact.bytes !== undefined) {
    return <Data>{bytes(fact.bytes)}</Data>;
  }
  if (fact.at !== undefined) {
    return <Data>{clock(fact.at)}</Data>;
  }
  return fact.text;
}

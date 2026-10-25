import type { HealthCategoryID, HealthFact, HealthItem } from "../api/gen/model";
import { bytes, clock } from "../format";
import { Data } from "../ui/data";
import { signalLook } from "../ui/signal-row";
import { ItemAction } from "./item-action";

export function ItemBody({ category, item }: { category: HealthCategoryID; item: HealthItem }) {
  const { dim } = signalLook[item.level];
  return (
    <div className="flex w-full flex-col items-start gap-3">
      {item.unsure && (
        <p className={dim}>
          SlipStream can&apos;t check this right now. This shows its last known state.
        </p>
      )}
      {item.detail.length > 0 && <Facts facts={item.detail} labels={dim} />}
      {item.action !== null && <ItemAction category={category} item={item} action={item.action} />}
    </div>
  );
}

function Facts({ facts, labels }: { facts: readonly HealthFact[]; labels: string }) {
  return (
    <dl className="grid w-full grid-cols-[max-content_minmax(0,1fr)] gap-x-4 gap-y-1 rounded-item bg-well px-3 py-2">
      {facts.map((fact) => (
        <div key={fact.label} className="contents">
          <dt className={labels}>{fact.label}</dt>
          <dd className="wrap-anywhere">
            <FactValue fact={fact} />
          </dd>
        </div>
      ))}
    </dl>
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

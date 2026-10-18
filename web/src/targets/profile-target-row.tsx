import { InformationCircleIcon } from "@hugeicons/core-free-icons";

import type { TargetsEstimate, TargetsTarget } from "../api/gen/model";
import { bytes } from "../format";
import { Icon } from "../ui/icon";
import { listed } from "./words";

function estimateText(estimate: TargetsEstimate | undefined): string {
  const total = estimate?.bytes ?? null;
  const rate = estimate?.mibPerMinute ?? null;
  if (total === null || rate === null) {
    return "No estimate yet";
  }
  return `About ${bytes(total)} across its Works, at ${String(Number(rate.toFixed(1)))} MiB per minute`;
}

// Every file of `id` also meets its containers, so `id` will rarely get a file of its own.
export const containersOf = (
  id: string,
  ids: readonly string[],
  byId: ReadonlyMap<string, TargetsTarget>,
): TargetsTarget[] =>
  ids.flatMap((other) => {
    const target = byId.get(other);
    return other !== id && target?.contains.includes(id) === true ? [target] : [];
  });

export function TargetRow({
  name,
  estimate,
  containers,
}: {
  name: string;
  estimate: TargetsEstimate | undefined;
  containers: readonly TargetsTarget[];
}) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5 py-1">
      <span className="font-medium text-ink-1">{name}</span>
      <span className="text-meta text-ink-3">{estimateText(estimate)}</span>
      {containers.length > 0 && (
        <span className="flex items-start gap-1.5 text-meta text-ink-2">
          <Icon icon={InformationCircleIcon} className="mt-px size-3.5 shrink-0" />
          Every file {name} takes also meets {listed(containers.map((c) => c.name))}.
        </span>
      )}
    </div>
  );
}

import type { GrabsGrab } from "../api/gen/model";
import { useListTargetsSuspense } from "../api/gen/query/targets/targets";
import { useQualityNames } from "../targets/quality-names";
import { Chip } from "../ui/chip";

type Job = { readonly grab: GrabsGrab; readonly work: string };

export function ForCell({ grab }: Job) {
  const targets = new Map(
    useListTargetsSuspense().data.targets.map((target) => [target.id, target.name]),
  );
  const quality = useQualityNames();
  return (
    <span className="flex flex-wrap gap-1">
      {grab.quality !== undefined && <Chip look="outline">{quality(grab.quality)}</Chip>}
      {grab.targetIds.map((id) => (
        <Chip key={id}>{targets.get(id) ?? "A deleted Target"}</Chip>
      ))}
    </span>
  );
}
export { JobMenu } from "./job-menu";

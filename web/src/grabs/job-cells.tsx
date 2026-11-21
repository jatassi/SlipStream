import type { GrabsGrab } from "../api/gen/model";
import { useListTargetsSuspense } from "../api/gen/query/targets/targets";
import { Quality } from "../release/quality";
import { Chip } from "../ui/chip";

type Job = { readonly grab: GrabsGrab; readonly work: string };

export function ForCell({ grab }: Job) {
  const targets = new Map(
    useListTargetsSuspense().data.targets.map((target) => [target.id, target.name]),
  );
  const { quality } = grab.releaseFacts;
  return (
    <span className="flex flex-wrap gap-1">
      {quality !== undefined && <Quality quality={quality} />}
      {grab.targetIds.map((id) => (
        <Chip key={id}>{targets.get(id) ?? "A deleted Target"}</Chip>
      ))}
    </span>
  );
}
export { JobMenu } from "./job-menu";

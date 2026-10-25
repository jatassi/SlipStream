import type { GrabsGrab } from "../api/gen/model";
import { activityStages } from "../shell/counts";
import { type StatusPart, StatusSentence } from "../ui/status-sentence";
import { failureAnchor } from "./failure-cards";
import type { FailedGrab } from "./stages";

const count = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

function statusParts(grabs: readonly GrabsGrab[], failed: readonly FailedGrab[]): StatusPart[] {
  const moving = grabs.filter((grab) => activityStages[grab.stage]).length;
  const imported = grabs.filter((grab) => grab.stage === "imported").length;
  const [firstFailed] = failed;
  return [
    moving === 0
      ? { kind: "plain", text: "Nothing moving" }
      : { kind: "live", text: count(moving, "job moving", "jobs moving") },
    ...(firstFailed === undefined
      ? []
      : [
          {
            kind: "failure",
            text: count(failed.length, "failure", "failures"),
            href: `#${failureAnchor(firstFailed)}`,
          } as const,
        ]),
    ...(imported === 0 ? [] : [{ kind: "plain", text: `${imported} imported` } as const]),
  ];
}

export function JobsStatus({
  grabs,
  failed,
}: {
  grabs: readonly GrabsGrab[];
  failed: readonly FailedGrab[];
}) {
  return <StatusSentence parts={statusParts(grabs, failed)} />;
}

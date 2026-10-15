import { Link } from "@tanstack/react-router";

import type { GrabsGrab } from "../api/gen/model";
import { activityStages } from "../shell/counts";
import { Empty } from "../shell/empty";
import { Button } from "../ui/button";
import { Section } from "../ui/section";
import { type StatusPart, StatusSentence } from "../ui/status-sentence";
import { GrabFailure, failureAnchor } from "./failure-cards";
import { GrabTable } from "./grab-table";
import { type FailedGrab, isFailed, standing } from "./stages";

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

export function ActivityBody({
  grabs: all,
  labels,
}: {
  grabs: readonly GrabsGrab[];
  labels: ReadonlyMap<string, string>;
}) {
  const grabs = standing(all);
  if (grabs.length === 0) {
    return (
      <Empty
        action={
          <Button nativeButton={false} render={<Link to="/library" />}>
            Open the Library
          </Button>
        }
      >
        Nothing is downloading. Open a Work and choose Search to find a release for it.
      </Empty>
    );
  }
  const work = (grab: GrabsGrab) => labels.get(grab.workId) ?? grab.releaseTitle;
  const failed = grabs.filter((grab) => isFailed(grab));
  const jobs = grabs.filter((grab) => !isFailed(grab));
  return (
    <>
      <div className="mb-6">
        <StatusSentence parts={statusParts(grabs, failed)} />
      </div>
      {failed.length > 0 && (
        <Section title="Needs you" count={failed.length}>
          <div className="grid grid-cols-[repeat(auto-fit,minmax(min(360px,100%),1fr))] gap-3">
            {failed.map((grab) => (
              <GrabFailure key={grab.id} grab={grab} work={work(grab)} form="full" />
            ))}
          </div>
        </Section>
      )}
      {jobs.length > 0 && (
        <Section title="Jobs" count={jobs.length}>
          <GrabTable layout="activity" rows={jobs.map((grab) => ({ grab, work: work(grab) }))} />
        </Section>
      )}
    </>
  );
}

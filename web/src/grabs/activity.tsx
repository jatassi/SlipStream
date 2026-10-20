import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

import type { GrabsGrab } from "../api/gen/model";
import { Empty } from "../shell/empty";
import { Button } from "../ui/button";
import { Section } from "../ui/section";
import { GrabFailure } from "./failure-cards";
import { GrabTable } from "./grab-table";
import { JobsStatus } from "./jobs-status";
import { isFailed, withoutRetriedFailures } from "./stages";

export function ActivityBody({
  grabs: all,
  labels,
  filter,
}: {
  grabs: readonly GrabsGrab[];
  labels: ReadonlyMap<string, string>;
  filter?: ReactNode;
}) {
  const grabs = withoutRetriedFailures(all);
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
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <JobsStatus grabs={grabs} failed={failed} />
        {filter}
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

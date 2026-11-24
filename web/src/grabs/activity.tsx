import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

import type { GrabsGrab, MovesBatchRow } from "../api/gen/model";
import { MovesSection } from "../moves/activity-band";
import { Button } from "../ui/button";
import { Empty } from "../ui/empty";
import { Section } from "../ui/section";
import { GrabFailure } from "./failure-cards";
import { GrabTable, useTorrentOf } from "./grab-table";
import { JobsStatus } from "./jobs-status";
// oxlint-disable-next-line import/max-dependencies -- Activity lays out its Needs you, Moves and Jobs sections and their empty state
import { type FailedGrab, needsYou, withoutReplacedFailures } from "./stages";

function Failed({
  grabs,
  work,
}: {
  grabs: readonly FailedGrab[];
  work: (grab: GrabsGrab) => string;
}) {
  return (
    <Section title="Needs you" count={grabs.length}>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(min(360px,100%),1fr))] gap-3">
        {grabs.map((grab) => (
          <GrabFailure
            key={grab.id}
            grab={grab}
            subject={work(grab)}
            work={work(grab)}
            form="full"
          />
        ))}
      </div>
    </Section>
  );
}

function JobsTable({
  jobs,
  work,
  labels,
}: {
  jobs: readonly GrabsGrab[];
  work: (grab: GrabsGrab) => string;
  labels: ReadonlyMap<string, string>;
}) {
  const torrentOf = useTorrentOf();
  const names = { of: (id: string) => labels.get(id), unknown: "A deleted Work" };
  return (
    <Section title="Jobs" count={jobs.length}>
      <GrabTable
        layout="activity"
        rows={jobs.map((grab) => ({ grab, work: work(grab), torrent: torrentOf(grab), names }))}
      />
    </Section>
  );
}

export function ActivityBody({
  grabs: all,
  moves,
  labels,
  filter,
}: {
  grabs: readonly GrabsGrab[];
  moves: readonly MovesBatchRow[];
  labels: ReadonlyMap<string, string>;
  filter?: ReactNode;
}) {
  const grabs = withoutReplacedFailures(all);
  if (grabs.length === 0 && moves.length === 0) {
    return (
      <Empty
        look="sheet"
        action={
          <Button nativeButton={false} render={<Link to="/library" />}>
            Open the Library
          </Button>
        }
      >
        Nothing is downloading. Open a Work and choose Search to find a Release for it.
      </Empty>
    );
  }
  const work = (grab: GrabsGrab) => labels.get(grab.workId) ?? grab.releaseTitle;
  const failed = grabs.filter((grab) => needsYou(grab));
  const jobs = grabs.filter((grab) => !needsYou(grab));
  return (
    <>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <JobsStatus
          grabs={grabs}
          failed={failed}
          moves={moves.filter(({ finishedAt }) => finishedAt === undefined).length}
        />
        {filter}
      </div>
      {failed.length > 0 && <Failed grabs={failed} work={work} />}
      <MovesSection batches={moves} />
      {jobs.length > 0 && <JobsTable jobs={jobs} work={work} labels={labels} />}
    </>
  );
}

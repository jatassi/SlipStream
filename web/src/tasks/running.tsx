import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useMemo } from "react";

import type { TasksOpenRun } from "../api/gen/model";
import { getListOpenTaskRunsQueryKey, listOpenTaskRuns } from "../api/gen/query/tasks/tasks";
import { useClockValue } from "../ui/clock";
import { Data } from "../ui/data";
import { List, ListRow } from "../ui/list";
import { ProgressCell, percentOf } from "../ui/progress";
import { Section } from "../ui/section";
// oxlint-disable-next-line import/max-dependencies -- the list reads the open runs and the clock, and draws each run's progress
import { CancelRun } from "./task-actions";
import { progressWords, visibleRuns } from "./words";

export function useVisibleRuns(): TasksOpenRun[] {
  const open = useQuery({
    queryKey: getListOpenTaskRunsQueryKey(),
    queryFn: ({ signal }) => listOpenTaskRuns({ signal }),
  }).data;
  // Reading which runs show, rather than the time, re-renders the shell only
  // when a held run comes due, since each render restarts a route still rendering.
  const mayHold = open?.runs.some((run) => run.trigger !== "run-now") === true;
  const shown = useClockValue(mayHold ? "second" : "minute", (now) =>
    open === undefined
      ? ""
      : visibleRuns(open, now)
          .map((run) => run.runId)
          .join(" "),
  );
  return useMemo(() => {
    const ids = new Set(shown.split(" "));
    return open?.runs.filter((run) => ids.has(String(run.runId))) ?? [];
  }, [open, shown]);
}

export const runMeterLabel = (run: TasksOpenRun): string => `${run.name} progress`;

function RunMeta({ run }: { run: TasksOpenRun }) {
  const nodes = run.nodes.join(", ");
  return (
    <span className="block pt-0.5">
      <ProgressCell
        label={runMeterLabel(run)}
        value={percentOf(run.done, run.total, 0)}
        motion="live"
        span="fill"
        amount={run.total > 0 ? <Data>{progressWords(run, run.noun)}</Data> : nodes}
        note={run.total > 0 ? nodes : undefined}
      />
    </span>
  );
}

const runRows = (runs: readonly TasksOpenRun[]) =>
  runs.map((run) => (
    <ListRow
      key={run.runId}
      title={
        <Link to="/tasks/$taskId" params={{ taskId: run.taskId }}>
          {run.name}
        </Link>
      }
      meta={<RunMeta run={run} />}
      trailing={<CancelRun runId={run.runId} name={run.name} />}
    />
  ));

export function RunningPanel({ runs }: { runs: readonly TasksOpenRun[] }) {
  return (
    <ul aria-label="Running tasks" className="-mx-4 -my-2">
      {runRows(runs)}
    </ul>
  );
}

export function RunningList() {
  const runs = useVisibleRuns();
  return (
    runs.length > 0 && (
      <div className="mb-7">
        <Section title="Running tasks">
          <List label="Running tasks">{runRows(runs)}</List>
        </Section>
      </div>
    )
  );
}

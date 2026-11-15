import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";

import type { TasksOpenRun } from "../api/gen/model";
import { getListOpenTaskRunsQueryKey, listOpenTaskRuns } from "../api/gen/query/tasks/tasks";
import { List, ListRow } from "../ui/list";
import { Meter } from "../ui/meter";
import { Section } from "../ui/section";
import { CancelRun } from "./task-actions";
import { percentOf, progressWords, visibleRuns } from "./words";

export function useVisibleRuns(): TasksOpenRun[] {
  const open = useQuery({
    queryKey: getListOpenTaskRunsQueryKey(),
    queryFn: ({ signal }) => listOpenTaskRuns({ signal }),
  }).data;
  const [now, setNow] = useState(() => new Date());
  const visible = open === undefined ? [] : visibleRuns(open, now);
  // A ticking clock re-renders the shell, which restarts any route still
  // rendering, so the clock only runs while the header holds a run back.
  const holding = open !== undefined && visible.length < open.runs.length;
  useEffect(() => {
    const timer = holding
      ? setInterval(() => {
          setNow(new Date());
        }, 1000)
      : undefined;
    return () => {
      clearInterval(timer);
    };
  }, [holding]);
  return visible;
}

export const runMeterLabel = (run: TasksOpenRun): string => `${run.name} progress`;

function RunMeta({ run }: { run: TasksOpenRun }) {
  const words = [run.nodes.join(", "), ...(run.total > 0 ? [progressWords(run, run.noun)] : [])];
  return (
    <span className="flex flex-col gap-1.5 pt-0.5">
      <span className="truncate">{words.filter(Boolean).join(" · ")}</span>
      <Meter label={runMeterLabel(run)} value={percentOf(run)} />
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

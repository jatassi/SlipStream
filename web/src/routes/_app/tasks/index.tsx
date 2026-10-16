import { createFileRoute } from "@tanstack/react-router";

import {
  getListOpenTaskRunsSuspenseQueryOptions,
  getListTasksSuspenseQueryOptions,
  useListOpenTaskRunsSuspense,
  useListTasksSuspense,
} from "../../../api/gen/query/tasks/tasks";
import { ensureQuery } from "../../../api/query-client";
import { Page } from "../../../shell/page";
import { Schedule } from "../../../tasks/schedule";
import { TaskGroup } from "../../../tasks/task-groups";
import { useNow } from "../../../tasks/use-now";
import { Skeleton } from "../../../ui/skeleton";
import { StatusSentence } from "../../../ui/status-sentence";

export const Route = createFileRoute("/_app/tasks/")({
  staticData: { title: "Tasks" },
  loader: ({ context }) =>
    Promise.all([
      ensureQuery(context.queryClient, getListTasksSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListOpenTaskRunsSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading Tasks" />,
  component: TasksPage,
});

const wordsTick = 15_000;

function TasksPage() {
  const { sentence, groups, multinode, settings } = useListTasksSuspense().data;
  const { runs } = useListOpenTaskRunsSuspense().data;
  const live = { now: useNow(wordsTick), progress: new Map(runs.map((run) => [run.runId, run])) };
  return (
    <Page>
      <div className="mb-6">
        <StatusSentence parts={sentence} />
      </div>
      {groups.map((group) => (
        <TaskGroup key={group.group} group={group} multinode={multinode} live={live} />
      ))}
      <Schedule settings={settings} />
    </Page>
  );
}

import { createFileRoute } from "@tanstack/react-router";

import {
  getListGrabsSuspenseQueryOptions,
  useListGrabsSuspense,
} from "../../../api/gen/query/grabs/grabs";
import {
  getListWorksSuspenseQueryOptions,
  useListWorksSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { ActivityBody } from "../../../grabs/activity";
import { RunningList } from "../../../tasks/running";
import { useShell } from "../../../ui/shell-context";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/activity/")({
  staticData: { title: "Activity" },
  loader: ({ context }) =>
    Promise.all([
      ensureQuery(context.queryClient, getListGrabsSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListWorksSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading Jobs" />,
  component: Jobs,
});

function Jobs() {
  const { grabs } = useListGrabsSuspense().data;
  const labels = new Map(useListWorksSuspense().data.works.map((work) => [work.id, work.label]));
  return (
    <>
      {useShell() === "phone" && <RunningList />}
      <ActivityBody grabs={grabs} labels={labels} />
    </>
  );
}

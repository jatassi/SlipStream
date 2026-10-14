import { Link, createFileRoute } from "@tanstack/react-router";

import {
  getListGrabsSuspenseQueryOptions,
  useListGrabsSuspense,
} from "../../api/gen/query/grabs/grabs";
import {
  getListWorksSuspenseQueryOptions,
  useListWorksSuspense,
} from "../../api/gen/query/works/works";
import { ensureQuery } from "../../api/query-client";
import { GrabStatus } from "../../grabs/grab-status";
import { Skeleton } from "../../ui/skeleton";

export const Route = createFileRoute("/_app/activity")({
  loader: ({ context }) =>
    Promise.all([
      ensureQuery(context.queryClient, getListGrabsSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListWorksSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading Activity" />,
  component: Activity,
});

function Activity() {
  const { grabs } = useListGrabsSuspense().data;
  const labels = new Map(useListWorksSuspense().data.works.map((work) => [work.id, work.label]));
  return (
    <section className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">Activity</h1>
      {grabs.length === 0 ? (
        <p className="text-ink-3">No Grabs yet.</p>
      ) : (
        <ul aria-label="Grabs" className="flex flex-col gap-4">
          {grabs.map((grab) => (
            <li key={grab.id} className="flex flex-col gap-1">
              <Link to="/works/$workId" params={{ workId: grab.workId }} className="underline">
                {labels.get(grab.workId)}
              </Link>
              <span className="font-mono text-sm wrap-anywhere">{grab.releaseTitle}</span>
              <GrabStatus grab={grab} />
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

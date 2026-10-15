import { createFileRoute } from "@tanstack/react-router";

import {
  getGetSystemSuspenseQueryOptions,
  useGetSystemSuspense,
} from "../../api/gen/query/system/system";
import { ensureQuery } from "../../api/query-client";
import { Skeleton } from "../../ui/skeleton";
import { Data } from "../../ui/data";

export const Route = createFileRoute("/_app/system")({
  loader: ({ context }) => ensureQuery(context.queryClient, getGetSystemSuspenseQueryOptions()),
  pendingComponent: () => <Skeleton label="Loading This Node" />,
  component: SystemPage,
});

function SystemPage() {
  const { data } = useGetSystemSuspense();
  const rows = [
    ["Version", data.build.version ?? "unstamped"],
    ["Data dir", data.dataDir],
    ["HTTP", data.listeners.http],
    ["Torrent", data.listeners.torrent],
    ["Link", data.listeners.link],
    ["Control socket", data.listeners.control],
  ] as const;
  return (
    <section className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">This Node</h1>
      <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-2">
        {rows.map(([term, value]) => (
          <div key={term} className="contents">
            <dt className="text-ink-3">{term}</dt>
            <dd className="wrap-anywhere">
              <Data>{value}</Data>
            </dd>
          </div>
        ))}
      </dl>
    </section>
  );
}

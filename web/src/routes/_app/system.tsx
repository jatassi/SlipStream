import { createFileRoute } from "@tanstack/react-router";

import {
  getGetSystemSuspenseQueryOptions,
  useGetSystemSuspense,
} from "../../api/gen/query/system/system";
import { ensureQuery } from "../../api/query-client";
import { Page } from "../../shell/page";
import { Skeleton } from "../../ui/skeleton";
import { Data } from "../../ui/data";

export const Route = createFileRoute("/_app/system")({
  staticData: { title: "This Node" },
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
    <Page>
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
    </Page>
  );
}

import { createFileRoute } from "@tanstack/react-router";

import {
  getGetSystemSuspenseQueryOptions,
  useGetSystemSuspense,
} from "../../api/gen/query/system/system";
import { ensureQuery } from "../../api/query-client";
import { clock } from "../../format";
import { Page } from "../../shell/page";
import { Data } from "../../ui/data";
import { Section } from "../../ui/section";
import { Skeleton } from "../../ui/skeleton";

export const Route = createFileRoute("/_app/system")({
  staticData: { title: "This Node" },
  loader: ({ context }) => ensureQuery(context.queryClient, getGetSystemSuspenseQueryOptions()),
  pendingComponent: () => <Skeleton label="Loading This Node" />,
  component: SystemPage,
});

type Row = readonly [term: string, value: string];

function KeyValues({ rows }: { rows: readonly Row[] }) {
  return (
    <dl className="rounded-card bg-glass text-(length:--body-size) leading-(--body-line-height) ring-1 ring-hairline ring-inset">
      {rows.map(([term, value]) => (
        <div
          key={term}
          className="grid min-h-(--row-height) grid-cols-[minmax(120px,200px)_minmax(0,1fr)] items-center gap-x-4 px-4 py-2 not-first:shadow-[inset_0_1px_0_var(--color-hairline)] in-data-[density=touch]:grid-cols-1 in-data-[density=touch]:gap-y-0.5"
        >
          <dt className="text-ink-3">{term}</dt>
          <dd className="wrap-anywhere text-ink-1">
            <Data>{value}</Data>
          </dd>
        </div>
      ))}
    </dl>
  );
}

function SystemPage() {
  const { data } = useGetSystemSuspense();
  return (
    <Page>
      <Section title="Overview">
        <KeyValues
          rows={[
            ["Version", data.build.version ?? "unstamped"],
            ["Running since", clock(data.startedAt)],
            ["Data folder", data.dataDir],
          ]}
        />
      </Section>
      <Section title="Listening on">
        <KeyValues
          rows={[
            ["Web and API", data.listeners.http],
            ["Torrents", data.listeners.torrent],
            ["Node link", data.listeners.link],
            ["Control socket", data.listeners.control],
          ]}
        />
      </Section>
    </Page>
  );
}

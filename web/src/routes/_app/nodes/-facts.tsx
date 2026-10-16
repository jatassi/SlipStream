import { useNavigate } from "@tanstack/react-router";

import type { SystemReport } from "../../../api/gen/model";
import { useGetNodeSystemSuspense } from "../../../api/gen/query/system/system";
import { clock } from "../../../format";
import { Data } from "../../../ui/data";
import { Section } from "../../../ui/section";
import { WarningLine } from "../../../ui/warning-line";
import { portOf } from "./-node";

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

function Fallback({ nodeId, system }: { nodeId: string; system: SystemReport }) {
  const navigate = useNavigate();
  const configured = portOf(system.http.configured);
  const bound = portOf(system.listeners.http);
  if (configured === 0 || configured === bound) {
    return null;
  }
  return (
    <div className="mb-3">
      <WarningLine
        chip="Port taken"
        subject={`Port ${configured} was taken, so SlipStream is on ${bound}.`}
        detail={`Another program holds ${configured}. Free it, or choose another port and restart.`}
        action={{
          label: "Change port",
          onClick: () => {
            void navigate({ to: "/nodes/$nodeId/web", params: { nodeId } });
          },
        }}
      />
    </div>
  );
}

export function Facts({ nodeId }: { nodeId: string }) {
  const { data } = useGetNodeSystemSuspense(nodeId);
  return (
    <>
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
        <Fallback nodeId={nodeId} system={data} />
        <KeyValues
          rows={[
            ["Web and API", data.listeners.http],
            ["Torrents", data.listeners.torrent],
            ["Node link", data.listeners.link],
            ["Control socket", data.listeners.control],
          ]}
        />
      </Section>
    </>
  );
}

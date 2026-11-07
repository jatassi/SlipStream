import type {
  DownloadclientsExternalClient as External,
  DownloadclientsKind as Kind,
} from "../api/gen/model";
import { useListDownloadClientsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { ClientSheetButton } from "../download-clients/client-sheet";
import { kindOf } from "../download-clients/client-form";
import { statusOf } from "../download-clients/client-status";
import { type Clock, countWords, sinceWords } from "../format";
import { SignalChip } from "../ui/signal-chip";
import { CheckNow } from "./client-check";
import type { Row } from "./model";

const torrents = { one: "torrent", many: "torrents" } as const;

function staleWords(count: number): string | undefined {
  if (count === 0) {
    return undefined;
  }
  return `${countWords(count, torrents)} ${count === 1 ? "shows" : "show"} last known data.`;
}

type CardProps = {
  client: External;
  node: string;
  kinds: readonly Kind[];
  nodes: readonly { readonly id: string; readonly name: string }[];
  stale: number;
  clock: Clock;
};

function ClientCard({ client, node, kinds, nodes, stale, clock }: CardProps) {
  const subject = `${client.name} on ${node}`;
  const status = statusOf(client, kindOf(kinds, client.kind), node);
  const since = client.since === null ? undefined : `Since ${sinceWords(client.since, clock)}`;
  return (
    <section
      aria-label={subject}
      className="flex flex-col gap-2 rounded-card bg-fail-card px-5 py-3 text-(length:--body-size) leading-(--body-line-height) text-fail-text ring-1 ring-fail-line ring-inset"
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <SignalChip kind="fail">{status.chip}</SignalChip>
        <span className="font-bold">{subject}</span>
        {since !== undefined && <span className="text-fail-dim">{since}</span>}
      </div>
      {status.sentence !== undefined && <p>{status.sentence}</p>}
      {staleWords(stale) !== undefined && <p className="text-fail-dim">{staleWords(stale)}</p>}
      <CheckNow>
        <ClientSheetButton
          label="Edit client"
          look="on-fail"
          opened={{ kind: "edit", client }}
          kinds={kinds}
          nodes={nodes}
        />
      </CheckNow>
    </section>
  );
}

/**
 * One red card per external client that isn't responding, refused the
 * sign-in or needs attention, leading the Torrents view.
 */
export function ClientCards({ rows, clock }: { rows: readonly Row[]; clock: Clock }) {
  const { cards, kinds } = useListDownloadClientsSuspense().data;
  const nodes = cards.map((card) => ({ id: card.nodeId, name: card.nodeName }));
  const away = cards.flatMap((card) =>
    card.externals
      .filter((client) => client.away)
      .map((client) => ({ client, node: card.nodeName })),
  );
  if (away.length === 0) {
    return null;
  }
  return (
    <div className="mb-6 flex flex-col gap-3">
      {away.map(({ client, node }) => (
        <ClientCard
          key={client.id}
          client={client}
          node={node}
          kinds={kinds}
          nodes={nodes}
          stale={
            rows.filter((row) => row.clientId === client.id && row.lastKnown !== undefined).length
          }
          clock={clock}
        />
      ))}
    </div>
  );
}

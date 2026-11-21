import { useState } from "react";

import type {
  DownloadclientsExternalClient as External,
  DownloadclientsKind as Kind,
} from "../api/gen/model";
import { useListDownloadClientsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { ClientSheet } from "../download-clients/client-sheet";
import { kindOf } from "../download-clients/client-form";
import { statusOf } from "../download-clients/client-status";
import { type Clock, countWords, sinceWords } from "../format";
import { Callout } from "../ui/callout";
import { useCheckNow } from "./client-check";
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
  const [editing, setEditing] = useState(false);
  const check = useCheckNow();
  const subject = `${client.name} on ${node}`;
  const status = statusOf(client, kindOf(kinds, client.kind), node);
  return (
    <section aria-label={subject}>
      <Callout
        tone="fail"
        chip={status.chip}
        subject={subject}
        {...(client.since !== null && { since: sinceWords(client.since, clock) })}
        action={check.action}
        more={[
          {
            label: "Edit client",
            onClick: () => {
              setEditing(true);
            },
          },
        ]}
      >
        {status.sentence !== undefined && <p>{status.sentence}</p>}
        {staleWords(stale) !== undefined && <p>{staleWords(stale)}</p>}
        {check.answer}
      </Callout>
      <ClientSheet
        opened={{ kind: "edit", client }}
        kinds={kinds}
        nodes={nodes}
        open={editing}
        onOpenChange={setEditing}
      />
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

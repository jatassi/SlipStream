import { useId } from "react";

import type {
  DownloadclientsClientCard as Card,
  DownloadclientsKind as Kind,
} from "../api/gen/model";
import { Chip } from "../ui/chip";
import { BuiltInClient } from "./built-in-client";
import type { NodeChoice } from "./client-fields";
import { ClientSheetButton } from "./client-sheet";
import { ExternalRow } from "./external-row";
import { GrabsTo } from "./grabs-to";

/**
 * A downloading Node's card: where its new Grabs go, its built-in client,
 * then its external clients and the button that adds one.
 */
export function NodeCard({
  card,
  kinds,
  nodes,
}: {
  card: Card;
  kinds: readonly Kind[];
  nodes: readonly NodeChoice[];
}) {
  const titleId = useId();
  return (
    <section
      aria-labelledby={titleId}
      className="flex flex-col gap-4 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset"
    >
      <header className="flex flex-wrap items-start justify-between gap-3">
        <h3 id={titleId} className="flex items-center gap-2 font-semibold text-ink-1">
          <Chip look="outline">{card.nodeName}</Chip>
        </h3>
        {card.externals.some((client) => client.on) && <GrabsTo card={card} />}
      </header>
      <BuiltInClient card={card} />
      {card.externals.length > 0 && (
        <ul
          aria-label={`External clients on ${card.nodeName}`}
          className="-mx-4 border-t border-hairline"
        >
          {card.externals.map((client) => (
            <ExternalRow
              key={client.id}
              client={client}
              kinds={kinds}
              nodes={nodes}
              node={card.nodeName}
            />
          ))}
        </ul>
      )}
      <div>
        <ClientSheetButton
          label="Add a Download Client"
          opened={{ kind: "add", nodeId: card.nodeId }}
          kinds={kinds}
          nodes={nodes}
        />
      </div>
    </section>
  );
}

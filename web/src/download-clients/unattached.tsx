import { useId } from "react";

import type {
  DownloadclientsExternalClient as External,
  DownloadclientsKind as Kind,
  NodesNode,
} from "../api/gen/model";
import { countWords } from "../format";
import { Section } from "../ui/section";
import { kindOf } from "./client-form";
import { DeleteClientButton } from "./delete-client";
import { rowLook } from "./external-row";

const torrents = { one: "torrent", many: "torrents" } as const;

const about =
  "SlipStream doesn't check these clients, because their Nodes no longer hold Downloading. Delete each one, or give its Node Downloading again.";

const nodeName = (nodes: readonly NodesNode[], id: string): string =>
  nodes.find((node) => node.id === id)?.name ?? "a removed Node";

function UnattachedRow({ client, kind, node }: { client: External; kind: Kind; node: string }) {
  const nameId = useId();
  return (
    <li aria-labelledby={nameId} className={`${rowLook} sm:flex-row sm:items-center`}>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <h4 id={nameId} className="font-semibold text-ink-1">
          {client.name}
        </h4>
        <p className="text-meta text-ink-3">
          {kind.name} on {node}, {countWords(client.torrents, torrents)}
        </p>
      </div>
      <DeleteClientButton client={client} />
    </li>
  );
}

/** External clients whose Nodes no longer hold Downloading. SlipStream doesn't check them, and each can only be deleted. */
export function Unattached({
  orphans,
  kinds,
  nodes,
}: {
  orphans: readonly External[];
  kinds: readonly Kind[];
  nodes: readonly NodesNode[];
}) {
  return (
    <Section title="Nodes that no longer download">
      <p className="mb-3 text-ink-3">{about}</p>
      <ul className="rounded-card bg-glass ring-1 ring-hairline ring-inset">
        {orphans.map((client) => (
          <UnattachedRow
            key={client.id}
            client={client}
            kind={kindOf(kinds, client.kind)}
            node={nodeName(nodes, client.nodeId)}
          />
        ))}
      </ul>
    </Section>
  );
}

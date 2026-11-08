import { useId } from "react";

import type {
  DownloadclientsExternalClient as External,
  DownloadclientsKind as Kind,
} from "../api/gen/model";
import { countWords } from "../format";
import type { NodeChoice } from "./client-fields";
import { kindOf } from "./client-form";
import { ClientSheetButton } from "./client-sheet";
import { DeleteClientButton } from "./delete-client";
import { StatusLine } from "./status-line";

const torrents = { one: "torrent", many: "torrents" } as const;

export const rowLook =
  "relative flex flex-col gap-2 px-4 py-3 not-first:before:absolute not-first:before:inset-x-0 not-first:before:top-0 not-first:before:h-px not-first:before:bg-hairline in-data-[density=touch]:not-first:before:left-4";

const linkLook = "text-ink-2 underline hover:text-ink-1";

/** An external client on its Node's card: its status in words, its torrents, a link to its own web UI, Edit and Delete. */
export function ExternalRow({
  client,
  kinds,
  nodes,
  node,
}: {
  client: External;
  kinds: readonly Kind[];
  nodes: readonly NodeChoice[];
  node: string;
}) {
  const nameId = useId();
  const kind = kindOf(kinds, client.kind);
  return (
    <li aria-labelledby={nameId} className={rowLook}>
      <div className="flex flex-wrap items-baseline gap-x-2">
        <h4 id={nameId} className="font-semibold text-ink-1">
          {client.name}
        </h4>
        <p className="text-meta text-ink-3">
          {kind.name}, {countWords(client.torrents, torrents)}
        </p>
      </div>
      <StatusLine client={client} kind={kind} node={node} />
      <div className="flex flex-wrap items-center gap-2">
        {client.webUi !== null && (
          <a href={client.webUi} target="_blank" rel="noreferrer" className={`mr-2 ${linkLook}`}>
            Open {client.name}
          </a>
        )}
        <ClientSheetButton
          label="Edit"
          opened={{ kind: "edit", client }}
          kinds={kinds}
          nodes={nodes}
        />
        <DeleteClientButton client={client} />
      </div>
    </li>
  );
}

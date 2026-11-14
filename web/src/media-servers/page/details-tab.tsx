import type { ReactNode } from "react";

import type {
  MediaserversHolderView as Holder,
  MediaserversKindView as Kind,
  MediaserversServerView as Server,
} from "../../api/gen/model";
import {
  useListMediaServerCredentialHoldersSuspense,
  useListMediaServerKindsSuspense,
} from "../../api/gen/query/mediaservers/mediaservers";
import { agoWords } from "../../format";
import { KeyValues } from "../../routes/_app/nodes/-facts";
import { useNow } from "../../tasks/use-now";
import { kindOf } from "../server-form";

const fallbackWords: { readonly [F in Server["fallback"]]: string } = {
  available: "The Controller reaches it",
  none: "The Controller can't reach it",
  untested: "Not tested",
};

const minute = 60_000;

const words = (text: string) => <span>{text}</span>;

function Holders({ holders, kind }: { holders: readonly Holder[]; kind: Kind }) {
  return (
    <span className="flex flex-col gap-1">
      <span>
        {holders
          .map((holder) => (holder.current ? holder.node.name : `${holder.node.name} (before)`))
          .join(", ")}
      </span>
      {kind.signIn === "api_key_or_admin" && (
        <span className="text-meta text-ink-3">
          {`To take the key from these Nodes, replace it on ${kind.label} and paste the new key under Settings.`}
        </span>
      )}
    </span>
  );
}

/** The machinery behind the server's state: its addresses, version, routes and who holds its key. */
export function DetailsTab({ server }: { server: Server }) {
  const kind = kindOf(useListMediaServerKindsSuspense().data.kinds, server.kind);
  const { holders } = useListMediaServerCredentialHoldersSuspense(server.id).data;
  const now = useNow(minute);
  const when = (iso: string | undefined) => (iso === undefined ? "Never" : agoWords(iso, now));
  const rows: (readonly [string, ReactNode])[] = [
    ["Server", words(`${kind.label} ${server.version}`)],
    ["Address", server.address],
    ["Fallback address", server.fallbackAddress ?? words("The address")],
    ...(kind.browser ? [["Browser address", server.browserAddress ?? words("None")] as const] : []),
    ["Reached from", words(server.reachingNode.name)],
    ["Runs on", words(server.runsOnNode?.name ?? "No Node")],
    ["Fallback", words(fallbackWords[server.fallback])],
    ["Last contact", words(when(server.lastContactAt))],
    ["Last send", words(when(server.lastSendAt))],
    [
      "Key held by",
      holders.length === 0 ? (
        words("No Node")
      ) : (
        <Holders key="holders" holders={holders} kind={kind} />
      ),
    ],
    ["Server Id", server.identity],
  ];
  return <KeyValues rows={rows} />;
}

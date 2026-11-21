import type {
  MediaserversHolderView as Holder,
  MediaserversKindView as Kind,
  MediaserversServerView as Server,
} from "../../api/gen/model";
import {
  useListMediaServerCredentialHoldersSuspense,
  useListMediaServerKindsSuspense,
} from "../../api/gen/query/mediaservers/mediaservers";
import { Data } from "../../ui/data";
import { Fact, Facts } from "../../ui/facts";
import { RelativeTime } from "../../ui/relative-time";
import { kindOf } from "../server-form";

const fallbackWords: { readonly [F in Server["fallback"]]: string } = {
  available: "The Controller reaches it",
  none: "The Controller can't reach it",
  untested: "Not tested",
};

function Holders({ holders, kind }: { holders: readonly Holder[]; kind: Kind }) {
  return (
    <span className="flex flex-col gap-1">
      <span>
        {holders
          .map((holder) => (holder.current ? holder.node.name : `${holder.node.name} (earlier)`))
          .join(", ")}
      </span>
      {kind.signIn === "api_key_or_admin" && (
        <span className="text-meta text-ink-3">
          {`To revoke their copies, create a new key in ${kind.label} and paste it under Settings.`}
        </span>
      )}
    </span>
  );
}

const when = (iso: string | undefined) =>
  iso === undefined ? "Never" : <RelativeTime at={iso} form="age" />;

/** The machinery behind the server's state: its addresses, version, routes and who holds its key. */
export function DetailsTab({ server }: { server: Server }) {
  const kind = kindOf(useListMediaServerKindsSuspense().data.kinds, server.kind);
  const { holders } = useListMediaServerCredentialHoldersSuspense(server.id).data;
  return (
    <Facts look="sheet">
      <Fact label="Server">{`${kind.label} ${server.version}`}</Fact>
      <Fact label="Address">
        <Data>{server.address}</Data>
      </Fact>
      <Fact label="Fallback address">
        {server.fallbackAddress === undefined ? (
          "Same as Address"
        ) : (
          <Data>{server.fallbackAddress}</Data>
        )}
      </Fact>
      {kind.browser && (
        <Fact label="Browser address">
          {server.browserAddress === undefined ? "None" : <Data>{server.browserAddress}</Data>}
        </Fact>
      )}
      <Fact label="Connects from">{server.reachingNode.name}</Fact>
      <Fact label="Runs on">{server.runsOnNode?.name ?? "No Node"}</Fact>
      <Fact label="Fallback">{fallbackWords[server.fallback]}</Fact>
      <Fact label="Last contact">{when(server.lastContactAt)}</Fact>
      <Fact label="Last send">{when(server.lastSendAt)}</Fact>
      <Fact label="Nodes with the key">
        {holders.length === 0 ? "No Node" : <Holders holders={holders} kind={kind} />}
      </Fact>
      <Fact label="Server Id">
        <Data>{server.identity}</Data>
      </Fact>
    </Facts>
  );
}

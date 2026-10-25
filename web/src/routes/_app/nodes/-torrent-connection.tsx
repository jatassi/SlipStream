import type { NodesNode } from "../../../api/gen/model";
import { useGetNodeConfigSuspense } from "../../../api/gen/query/system/system";
import { Empty } from "../../../shell/empty";
import { askRestart } from "../../../shell/restart/restart-state";
import { Details } from "../../../ui/details";
import { Section } from "../../../ui/section";
import { WarningLine } from "../../../ui/warning-line";
import { NodeSettings, fieldsOf } from "./-config";

function PendingRestart({ node }: { node: NodesNode }) {
  const pending = fieldsOf(useGetNodeConfigSuspense(node.id).data, "torrent").filter(
    (field) => field.pendingRestart,
  );
  if (pending.length === 0) {
    return null;
  }
  return (
    <WarningLine
      signal="warn"
      chip="Restart needed"
      subject={`Restart ${node.name} to use the new ${pending.map((field) => field.label.toLowerCase()).join(" and ")}`}
      detail={pending
        .map((field) =>
          field.kind === "address"
            ? `It listens on ${field.running} until then.`
            : "Torrents already added stay where they are.",
        )
        .join(" ")}
      action={{
        label: "Restart now",
        onClick: () => {
          askRestart(node);
        },
      }}
    />
  );
}

const adapterNote =
  "Trackers reachable only through another network adapter, such as a VPN or a second network card, can't be reached from that address. To reach them, set the listen address to that adapter's address.";

/** The Node's own torrent rows, read at start: its listen address and downloads folder. */
export function TorrentConnection({ node, online }: { node: NodesNode; online: boolean }) {
  return (
    <Section title="Connection">
      {online ? (
        <div className="flex flex-col gap-3">
          <PendingRestart node={node} />
          <NodeSettings nodeId={node.id} section="torrent" />
          <Details summary="With 0.0.0.0 or [::], the client listens on the address this computer reaches the internet from.">
            <p className="mt-1 text-pretty">{adapterNote}</p>
          </Details>
        </div>
      ) : (
        <Empty>
          {node.name} is offline. Its listen address and downloads folder show here when it
          reconnects.
        </Empty>
      )}
    </Section>
  );
}

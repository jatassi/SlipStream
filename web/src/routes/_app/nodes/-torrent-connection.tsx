import type { NodesNode } from "../../../api/gen/model";
import { PendingRestart } from "../../../nodes/node-setting";
import { Details } from "../../../ui/details";
import { Empty } from "../../../ui/empty";
import { Section } from "../../../ui/section";
import { NodeSettings } from "./-config";

const adapterNote =
  "The client can't connect to a tracker that only another network adapter can get to, such as a VPN or a second network card. To use such a tracker, set the listen address to that adapter's address.";

/** The Node's own torrent rows, read at start: its listen address and downloads folder. */
export function TorrentConnection({ node, online }: { node: NodesNode; online: boolean }) {
  return (
    <Section title="Connection">
      {online ? (
        <div className="flex flex-col gap-3">
          <PendingRestart node={node} />
          <NodeSettings nodeId={node.id} section="torrent" />
          <Details summary="With 0.0.0.0 or [::], the client listens on every network address of this computer and connects out through its default network adapter.">
            <p className="mt-1 text-pretty">{adapterNote}</p>
          </Details>
        </div>
      ) : (
        <Empty look="sheet">
          {node.name} is offline. Its listen address and downloads folder show here when it
          reconnects.
        </Empty>
      )}
    </Section>
  );
}

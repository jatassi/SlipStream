import type { NodesNode } from "../api/gen/model";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListTransferSpeedsSuspense } from "../api/gen/query/transfers/transfers";
import { clock } from "../format";
import { rateWords } from "../torrents/model";
import { Data } from "../ui/data";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";

/** The last measured speed to and from each other Node. */
export function Speeds({ node }: { node: NodesNode }) {
  const { speeds } = useListTransferSpeedsSuspense().data;
  const { nodes } = useListNodesSuspense().data;
  const rows = speeds
    .flatMap((speed) => {
      const sends = speed.sourceNodeId === node.id;
      const peerId = sends ? speed.destinationNodeId : speed.sourceNodeId;
      const peer = nodes.find((other) => other.id === peerId);
      if ((!sends && speed.destinationNodeId !== node.id) || peer === undefined) {
        return [];
      }
      return [{ speed, title: sends ? `To ${peer.name}` : `From ${peer.name}` }];
    })
    .toSorted((a, b) => a.title.localeCompare(b.title));
  return (
    <Section title="Recent speeds">
      {rows.length === 0 ? (
        <p className="text-ink-3">
          No Transfer has moved bytes between {node.name} and another Node yet.
        </p>
      ) : (
        <List label={`Recent speeds of ${node.name}`}>
          {rows.map(({ speed, title }) => (
            <ListRow
              key={title}
              title={title}
              meta={`Measured ${clock(speed.rateAt)}`}
              trailing={<Data className="text-ink-1">{rateWords(speed.rate)}</Data>}
            />
          ))}
        </List>
      )}
    </Section>
  );
}

import type { NodesNode } from "../api/gen/model";
import {
  useListMediaServerKindsSuspense,
  useListMediaServersSuspense,
} from "../api/gen/query/mediaservers/mediaservers";
import { Empty } from "../shell/empty";
import { List } from "../ui/list";
import { Section } from "../ui/section";
import { ServerRow } from "./server-row";

/** The servers a Node reaches, each with its state, which leads with what to do. */
export function NodeServers({ node }: { node: NodesNode }) {
  const reached = useListMediaServersSuspense().data.servers.filter(
    (server) => server.reachingNode.id === node.id,
  );
  const { kinds } = useListMediaServerKindsSuspense().data;
  return (
    <Section title="Media Servers" count={reached.length}>
      {reached.length === 0 ? (
        <Empty>
          {`No Media Server connects from ${node.name}. To use one from ${node.name}, edit it in Settings > Media Servers and set Connect from to ${node.name}.`}
        </Empty>
      ) : (
        <List label={`Media Servers that connect from ${node.name}`}>
          {reached.map((server) => (
            <ServerRow key={server.id} server={server} kinds={kinds} />
          ))}
        </List>
      )}
    </Section>
  );
}

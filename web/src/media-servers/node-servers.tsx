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
          {`No Media Server is reached from ${node.name}. Choose ${node.name} under Reached from when you add or edit a server in Settings > Media Servers.`}
        </Empty>
      ) : (
        <List label={`Media Servers reached from ${node.name}`}>
          {reached.map((server) => (
            <ServerRow key={server.id} server={server} kinds={kinds} />
          ))}
        </List>
      )}
    </Section>
  );
}

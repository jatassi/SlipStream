import { Link } from "@tanstack/react-router";

import type { ReachabilityClassLine } from "../api/gen/model";
import { List, ListRow } from "../ui/list";

/** How each Node connects to the internet, in no signal colour, each opening its Network section. */
export function ClassLines({ nodes }: { nodes: readonly ReachabilityClassLine[] }) {
  return (
    <List label="Each Node's network">
      {nodes.map((node) => (
        <li key={node.nodeId}>
          <ListRow
            title={node.line}
            wrap
            render={<Link to="/nodes/$nodeId/network" params={{ nodeId: node.nodeId }} />}
          />
        </li>
      ))}
    </List>
  );
}

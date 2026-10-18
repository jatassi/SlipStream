import { useQuery } from "@tanstack/react-query";

import { isProblem } from "../../../../api/errors";
import type { NodesNode } from "../../../../api/gen/model";
import { getGetNodeSystemQueryKey, getNodeSystem } from "../../../../api/gen/query/system/system";
import { Empty } from "../../../../shell/empty";
import { Section } from "../../../../ui/section";
import { Skeleton } from "../../../../ui/skeleton";
import { Facts } from "../-facts";

/** The Node's own report, which a Worker gives only while it is online. */
export function System({ node }: { node: NodesNode }) {
  const system = useQuery({
    queryKey: getGetNodeSystemQueryKey(node.id),
    queryFn: ({ signal }) => getNodeSystem(node.id, { signal }),
  });
  if (system.isPending) {
    return <Skeleton label="Loading the Node's system" />;
  }
  if (system.isError) {
    const failure: unknown = system.error;
    if (!isProblem(failure, "node_offline")) {
      throw failure;
    }
    return (
      <Section title="System">
        <Empty>
          {node.name} is offline. Its version and listeners show again when it reconnects.
        </Empty>
      </Section>
    );
  }
  return <Facts nodeId={node.id} data={system.data} />;
}

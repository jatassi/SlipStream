import { useState } from "react";

import { useListNodesSuspense } from "../../../../api/gen/query/nodes/nodes";
import { useNow } from "../../../../tasks/use-now";
import { type Acting, NodeDialogs } from "../-manage/dialogs";
import { nodeOf } from "../-node";
import { NodeFacts } from "./facts";
import { RemoveRow } from "./remove-row";
import { System } from "./system";

/** A Node's Overview: what it is and does, how it is reached, its own report, and Remove. */
export function Overview({ nodeId }: { nodeId: string }) {
  const { nodes } = useListNodesSuspense().data;
  const node = nodeOf(nodes, nodeId);
  const now = useNow(60_000);
  const [acting, setActing] = useState<Acting>();
  return (
    <>
      <NodeFacts node={node} nodes={nodes} clock={{ now, zone: undefined }} act={setActing} />
      <System node={node} />
      <RemoveRow node={node} act={setActing} />
      <NodeDialogs
        acting={acting}
        onClose={() => {
          setActing(undefined);
        }}
      />
    </>
  );
}

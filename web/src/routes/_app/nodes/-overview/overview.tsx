import { useState } from "react";

import { useListNodesSuspense } from "../../../../api/gen/query/nodes/nodes";
import { type Acting, NodeDialogs } from "../-manage/dialogs";
import { nodeOf } from "../-node";
import { NodeFacts } from "./facts";
import { RemoveRow } from "./remove-row";
import { StopRow } from "./stop-row";
import { System } from "./system";

/** A Node's Overview: what it is and does, how it is reached, its own report, and Remove. */
export function Overview({ nodeId }: { nodeId: string }) {
  const { nodes } = useListNodesSuspense().data;
  const node = nodeOf(nodes, nodeId);
  const [acting, setActing] = useState<Acting>();
  return (
    <>
      <NodeFacts node={node} nodes={nodes} act={setActing} />
      <System node={node} />
      <StopRow node={node} />
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

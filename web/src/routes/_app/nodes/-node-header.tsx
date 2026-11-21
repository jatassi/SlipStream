import type { NodesNode } from "../../../api/gen/model";
import { useGetRoleBoardSuspense } from "../../../api/gen/query/nodes/nodes";
import { NodeHeading } from "../../../nodes/node-heading";
import type { NodeState } from "../../../nodes/state";
import { useClock } from "../../../ui/clock";
import { roleChips } from "./-landing/view";

/** A Node's heading, kept current to the minute apart from the section under it. */
export function NodeHeader({
  node,
  nodes,
  state,
}: {
  node: NodesNode;
  nodes: readonly NodesNode[];
  state: NodeState;
}) {
  const board = useGetRoleBoardSuspense().data;
  const clock = useClock();
  const away = state !== "online" && state !== "restarting";
  return (
    <NodeHeading
      state={state}
      seenAt={away ? node.link.since : undefined}
      roles={roleChips(
        { nodes },
        board,
        node.roles,
        { standing: node.standing, since: node.link.since },
        clock,
      )}
    />
  );
}

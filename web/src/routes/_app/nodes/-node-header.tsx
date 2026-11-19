import type { NodesNode } from "../../../api/gen/model";
import { useGetRoleBoardSuspense } from "../../../api/gen/query/nodes/nodes";
import { sinceWords } from "../../../format";
import { NodeHeading } from "../../../nodes/node-heading";
import type { NodeState } from "../../../nodes/state";
import { useNow } from "../../../tasks/use-now";
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
  const clock = { now: useNow(60_000), zone: undefined };
  const away = state !== "online" && state !== "restarting";
  return (
    <NodeHeading
      state={state}
      seen={away ? `Last seen ${sinceWords(node.link.since, clock)}` : undefined}
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

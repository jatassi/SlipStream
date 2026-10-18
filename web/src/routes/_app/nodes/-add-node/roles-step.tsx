import { useQuery, useQueryClient } from "@tanstack/react-query";

import type { NodesNode } from "../../../../api/gen/model";
import {
  getGetRoleBoardQueryKey,
  getListNodesQueryKey,
  getRoleBoard,
} from "../../../../api/gen/query/nodes/nodes";
import { refresh } from "../../../../live/refresh";
import { Pick } from "./pick";
import { Waiting } from "./waiting";

/** The new Node's Roles, once the role board seats it, through the Review. */
export function RolesStep({
  node,
  onClose,
  onApplied,
}: {
  node: NodesNode;
  onClose: () => void;
  onApplied: () => void;
}) {
  const queryClient = useQueryClient();
  const board = useQuery({
    queryKey: getGetRoleBoardQueryKey(),
    queryFn: ({ signal }) => getRoleBoard({ signal }),
  }).data;
  const seat = board?.nodes.find(({ id }) => id === node.id);
  if (board === undefined || seat === undefined) {
    return <Waiting text={`Loading the Roles of ${node.name}…`} />;
  }
  return (
    <Pick
      board={board}
      seat={seat}
      node={node}
      onClose={onClose}
      onApplied={async () => {
        await Promise.all([
          refresh(queryClient, { queryKey: getGetRoleBoardQueryKey() }),
          refresh(queryClient, { queryKey: getListNodesQueryKey() }),
        ]);
        onApplied();
      }}
    />
  );
}

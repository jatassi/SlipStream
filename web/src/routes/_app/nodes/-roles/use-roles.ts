import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import {
  getGetRoleBoardQueryKey,
  getListNodesQueryKey,
  useGetRoleBoardSuspense,
} from "../../../../api/gen/query/nodes/nodes";
import { refresh } from "../../../../live/refresh";
import { type Draft, draftOf, sameDraft, seatOf } from "../../../../nodes/roles";
import { useNow } from "../../../../tasks/use-now";
import { useNode } from "../-node";

export type Roles = ReturnType<typeof useRoles>;

/** One Node's place on the role board, the Roles ticked on it so far, and a reload once they change. */
export function useRoles(nodeId: string) {
  const board = useGetRoleBoardSuspense().data;
  const node = useNode(nodeId);
  const seat = seatOf(board, nodeId);
  const queryClient = useQueryClient();
  const now = useNow(60_000);
  const saved = draftOf(board, seat);
  const [edits, setEdits] = useState<Draft>();
  const draft = edits ?? saved;
  const reload = async () => {
    await Promise.all([
      refresh(queryClient, { queryKey: getGetRoleBoardQueryKey() }),
      refresh(queryClient, { queryKey: getListNodesQueryKey() }),
    ]);
  };
  return {
    board,
    node,
    seat,
    clock: { now, zone: undefined },
    draft,
    changed: !sameDraft(draft, saved),
    setDraft: setEdits,
    reload,
  };
}

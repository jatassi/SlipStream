import { useQueryClient } from "@tanstack/react-query";
import { type ReactNode, useState } from "react";

import type { NodesRoleBoard } from "../../../../api/gen/model";
import {
  getGetRoleBoardQueryKey,
  getGetTopologyQueryKey,
  getListNodesQueryKey,
  useGetRoleBoardSuspense,
} from "../../../../api/gen/query/nodes/nodes";
import type { Drop, GestureOf } from "../../../../diagram/drag";
import { refresh } from "../../../../live/refresh";
import { assignmentOf, draftOf, seatOf, withRole } from "../../../../nodes/roles";
import type { Change } from "../-roles/plan";
import { ReviewDialog } from "../-roles/review";

/** The change a dropped chip asks for: the Role off the Node it left unless ⌥ added it, and onto the Node it landed on. */
export function changeOf(board: NodesRoleBoard, { role, from, to, add }: Drop): Change {
  const onto = assignmentOf(to, withRole(draftOf(board, seatOf(board, to)), role, true));
  const off = assignmentOf(from, withRole(draftOf(board, seatOf(board, from)), role, false));
  return { basis: board.basis, set: add ? [onto] : [off, onto] };
}

/**
 * The plan's verdict on each drop the Diagram can make, from the role
 * board's gestures, and the Review a drop opens.
 */
export function useDrops(): {
  gestureOf: GestureOf;
  onDrop: (drop: Drop) => void;
  review: ReactNode;
} {
  const board = useGetRoleBoardSuspense().data;
  const queryClient = useQueryClient();
  const [reviewing, setReviewing] = useState<Change>();
  const close = () => {
    setReviewing(undefined);
  };
  return {
    gestureOf: (role, from, to, add) =>
      board.gestures.find(
        (gesture) =>
          gesture.role === role &&
          gesture.to === to &&
          (add ? gesture.from === undefined : gesture.from === from),
      ),
    onDrop: (drop) => {
      setReviewing(changeOf(board, drop));
    },
    review: reviewing !== undefined && (
      <ReviewDialog
        change={reviewing}
        onClose={close}
        onApplied={async () => {
          await Promise.all(
            [getGetRoleBoardQueryKey(), getGetTopologyQueryKey(), getListNodesQueryKey()].map(
              (queryKey) => refresh(queryClient, { queryKey }),
            ),
          );
          close();
        }}
      />
    ),
  };
}

import { getGetMoveQueryKey, getListMovesQueryKey } from "../api/gen/query/moves/moves";
import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import type { LiveEffects } from "./types";

/** A change to Moves reads the batch list again, each named batch's units and each named Work's view. */
export const moveEffects = {
  "moves.changed": ({ batchIds, workIds }, { refresh }) => {
    refresh({ queryKey: getListMovesQueryKey() });
    for (const id of batchIds) {
      refresh({ queryKey: getGetMoveQueryKey(id) });
    }
    for (const id of workIds) {
      refresh({ queryKey: getGetWorkQueryKey(id) });
    }
  },
} satisfies Pick<LiveEffects, "moves.changed">;

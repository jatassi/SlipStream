import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import type { LiveEffects } from "./types";

export const moveEffects = {
  "moves.changed": (data, { refresh }) => {
    for (const id of data.workIds) {
      refresh({ queryKey: getGetWorkQueryKey(id) });
    }
  },
} satisfies Pick<LiveEffects, "moves.changed">;

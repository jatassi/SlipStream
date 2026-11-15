import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import type { LiveEffects } from "./types";

/** What the search queue is doing for a Work shows on that Work, so a queue change reads each named Work again. */
export const searchEffects = {
  "grabs.queue-changed": ({ workIds }, { refresh }) => {
    for (const workId of workIds) {
      refresh({ queryKey: getGetWorkQueryKey(workId) });
    }
  },
} satisfies Pick<LiveEffects, "grabs.queue-changed">;

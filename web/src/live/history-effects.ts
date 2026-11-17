import { everyPath } from "./refresh";
import type { LiveEffects } from "./types";

const everyFeed = everyPath(/^\/api\/v1\/notifications\/feed$/u);

/** A sweep deletes bell rows with their entries, so the bell reads again. */
export const historyEffects = {
  "history.changed": (_data, { refresh }) => {
    refresh(everyFeed);
  },
} satisfies Pick<LiveEffects, "history.changed">;

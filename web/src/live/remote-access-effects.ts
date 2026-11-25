import { everyPath } from "./refresh";
import type { LiveEffects } from "./types";

const everyRemoteAccess = everyPath(/^\/api\/v1\/remote-access(\/|$)/u);

/** Remote access, its settings and each Node's Network section. */
export const remoteAccessEffects = {
  "reachability.changed": (_data, { refresh }) => {
    refresh(everyRemoteAccess);
  },
} satisfies Pick<LiveEffects, "reachability.changed">;

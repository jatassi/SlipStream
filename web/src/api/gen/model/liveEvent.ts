import type { LiveEventIdentitySessionsChanged } from "./liveEventIdentitySessionsChanged.ts";
import type { LiveEventLiveResync } from "./liveEventLiveResync.ts";
import type { LiveEventLiveSessionEnded } from "./liveEventLiveSessionEnded.ts";
import type { LiveEventNodesChanged } from "./liveEventNodesChanged.ts";

export type LiveEvent =
  | LiveEventIdentitySessionsChanged
  | LiveEventLiveResync
  | LiveEventLiveSessionEnded
  | LiveEventNodesChanged;

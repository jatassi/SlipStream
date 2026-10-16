import type { LiveEventFilesChanged } from "./liveEventFilesChanged.ts";
import type { LiveEventGrabsChanged } from "./liveEventGrabsChanged.ts";
import type { LiveEventGrabsSearchChanged } from "./liveEventGrabsSearchChanged.ts";
import type { LiveEventIdentitySessionsChanged } from "./liveEventIdentitySessionsChanged.ts";
import type { LiveEventIndexersChanged } from "./liveEventIndexersChanged.ts";
import type { LiveEventLiveResync } from "./liveEventLiveResync.ts";
import type { LiveEventLiveSessionEnded } from "./liveEventLiveSessionEnded.ts";
import type { LiveEventNodesChanged } from "./liveEventNodesChanged.ts";
import type { LiveEventRootsChanged } from "./liveEventRootsChanged.ts";
import type { LiveEventTargetsChanged } from "./liveEventTargetsChanged.ts";
import type { LiveEventTasksChanged } from "./liveEventTasksChanged.ts";
import type { LiveEventTasksProgress } from "./liveEventTasksProgress.ts";
import type { LiveEventWorksChanged } from "./liveEventWorksChanged.ts";
import type { LiveEventWorksContentTypesChanged } from "./liveEventWorksContentTypesChanged.ts";

export type LiveEvent =
  | LiveEventFilesChanged
  | LiveEventGrabsChanged
  | LiveEventGrabsSearchChanged
  | LiveEventIdentitySessionsChanged
  | LiveEventIndexersChanged
  | LiveEventLiveResync
  | LiveEventLiveSessionEnded
  | LiveEventNodesChanged
  | LiveEventRootsChanged
  | LiveEventTargetsChanged
  | LiveEventTasksChanged
  | LiveEventTasksProgress
  | LiveEventWorksChanged
  | LiveEventWorksContentTypesChanged;

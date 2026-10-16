import type { LiveEventFilesChanged } from "./liveEventFilesChanged.ts";
import type { LiveEventGrabsChanged } from "./liveEventGrabsChanged.ts";
import type { LiveEventGrabsSearchChanged } from "./liveEventGrabsSearchChanged.ts";
import type { LiveEventIdentitySessionsChanged } from "./liveEventIdentitySessionsChanged.ts";
import type { LiveEventIndexersChanged } from "./liveEventIndexersChanged.ts";
import type { LiveEventLiveResync } from "./liveEventLiveResync.ts";
import type { LiveEventLiveSessionEnded } from "./liveEventLiveSessionEnded.ts";
import type { LiveEventNodesChanged } from "./liveEventNodesChanged.ts";
import type { LiveEventNodeSettingsChanged } from "./liveEventNodeSettingsChanged.ts";
import type { LiveEventRootsChanged } from "./liveEventRootsChanged.ts";
import type { LiveEventTargetsChanged } from "./liveEventTargetsChanged.ts";
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
  | LiveEventNodeSettingsChanged
  | LiveEventNodesChanged
  | LiveEventRootsChanged
  | LiveEventTargetsChanged
  | LiveEventWorksChanged
  | LiveEventWorksContentTypesChanged;

import type { LiveEventFilesChanged } from "./liveEventFilesChanged.ts";
import type { LiveEventGrabsChanged } from "./liveEventGrabsChanged.ts";
import type { LiveEventGrabsSearchChanged } from "./liveEventGrabsSearchChanged.ts";
import type { LiveEventIdentityAccountsChanged } from "./liveEventIdentityAccountsChanged.ts";
import type { LiveEventIdentityLinksChanged } from "./liveEventIdentityLinksChanged.ts";
import type { LiveEventIdentityPasskeysChanged } from "./liveEventIdentityPasskeysChanged.ts";
import type { LiveEventIdentitySessionsChanged } from "./liveEventIdentitySessionsChanged.ts";
import type { LiveEventIndexersChanged } from "./liveEventIndexersChanged.ts";
import type { LiveEventLiveAccessChanged } from "./liveEventLiveAccessChanged.ts";
import type { LiveEventLiveResync } from "./liveEventLiveResync.ts";
import type { LiveEventLiveSessionEnded } from "./liveEventLiveSessionEnded.ts";
import type { LiveEventMetadataChanged } from "./liveEventMetadataChanged.ts";
import type { LiveEventNodesChanged } from "./liveEventNodesChanged.ts";
import type { LiveEventNodeSettingsChanged } from "./liveEventNodeSettingsChanged.ts";
import type { LiveEventNodesTraffic } from "./liveEventNodesTraffic.ts";
import type { LiveEventRootsChanged } from "./liveEventRootsChanged.ts";
import type { LiveEventTargetsChanged } from "./liveEventTargetsChanged.ts";
import type { LiveEventTasksChanged } from "./liveEventTasksChanged.ts";
import type { LiveEventTasksProgress } from "./liveEventTasksProgress.ts";
import type { LiveEventVolumesChanged } from "./liveEventVolumesChanged.ts";
import type { LiveEventWorksChanged } from "./liveEventWorksChanged.ts";
import type { LiveEventWorksContentTypesChanged } from "./liveEventWorksContentTypesChanged.ts";
import type { LiveEventWorksTypeChanged } from "./liveEventWorksTypeChanged.ts";

export type LiveEvent =
  | LiveEventFilesChanged
  | LiveEventGrabsChanged
  | LiveEventGrabsSearchChanged
  | LiveEventIdentityAccountsChanged
  | LiveEventIdentityLinksChanged
  | LiveEventIdentityPasskeysChanged
  | LiveEventIdentitySessionsChanged
  | LiveEventIndexersChanged
  | LiveEventLiveAccessChanged
  | LiveEventLiveResync
  | LiveEventLiveSessionEnded
  | LiveEventMetadataChanged
  | LiveEventNodeSettingsChanged
  | LiveEventNodesChanged
  | LiveEventNodesTraffic
  | LiveEventRootsChanged
  | LiveEventTargetsChanged
  | LiveEventTasksChanged
  | LiveEventTasksProgress
  | LiveEventVolumesChanged
  | LiveEventWorksChanged
  | LiveEventWorksContentTypesChanged
  | LiveEventWorksTypeChanged;

import type { LiveEventBackupsChanged } from "./liveEventBackupsChanged.ts";
import type { LiveEventDownloadClientsChanged } from "./liveEventDownloadClientsChanged.ts";
import type { LiveEventFilesChanged } from "./liveEventFilesChanged.ts";
import type { LiveEventFilesMoved } from "./liveEventFilesMoved.ts";
import type { LiveEventFirstrunChanged } from "./liveEventFirstrunChanged.ts";
import type { LiveEventGrabsBlocklistChanged } from "./liveEventGrabsBlocklistChanged.ts";
import type { LiveEventGrabsChanged } from "./liveEventGrabsChanged.ts";
import type { LiveEventGrabsQueueChanged } from "./liveEventGrabsQueueChanged.ts";
import type { LiveEventGrabsSearchChanged } from "./liveEventGrabsSearchChanged.ts";
import type { LiveEventHealthChanged } from "./liveEventHealthChanged.ts";
import type { LiveEventHistoryChanged } from "./liveEventHistoryChanged.ts";
import type { LiveEventHomeSettingsChanged } from "./liveEventHomeSettingsChanged.ts";
import type { LiveEventIdentityAccountsChanged } from "./liveEventIdentityAccountsChanged.ts";
import type { LiveEventIdentityAddressChanged } from "./liveEventIdentityAddressChanged.ts";
import type { LiveEventIdentityLinksChanged } from "./liveEventIdentityLinksChanged.ts";
import type { LiveEventIdentityPasskeysChanged } from "./liveEventIdentityPasskeysChanged.ts";
import type { LiveEventIdentitySessionsChanged } from "./liveEventIdentitySessionsChanged.ts";
import type { LiveEventImportsChanged } from "./liveEventImportsChanged.ts";
import type { LiveEventIndexersCatalogChanged } from "./liveEventIndexersCatalogChanged.ts";
import type { LiveEventIndexersChanged } from "./liveEventIndexersChanged.ts";
import type { LiveEventIndexersRssChanged } from "./liveEventIndexersRssChanged.ts";
import type { LiveEventIndexersSettingsChanged } from "./liveEventIndexersSettingsChanged.ts";
import type { LiveEventIndexersStatusChanged } from "./liveEventIndexersStatusChanged.ts";
import type { LiveEventLiveAccessChanged } from "./liveEventLiveAccessChanged.ts";
import type { LiveEventLiveResync } from "./liveEventLiveResync.ts";
import type { LiveEventLiveSessionEnded } from "./liveEventLiveSessionEnded.ts";
import type { LiveEventMediaServersChanged } from "./liveEventMediaServersChanged.ts";
import type { LiveEventMediaServersItemsFound } from "./liveEventMediaServersItemsFound.ts";
import type { LiveEventMediaServersRecovered } from "./liveEventMediaServersRecovered.ts";
import type { LiveEventMediaServersSessionsChanged } from "./liveEventMediaServersSessionsChanged.ts";
import type { LiveEventMediaServersUnhealthy } from "./liveEventMediaServersUnhealthy.ts";
import type { LiveEventMetadataChanged } from "./liveEventMetadataChanged.ts";
import type { LiveEventMovesChanged } from "./liveEventMovesChanged.ts";
import type { LiveEventNamingChanged } from "./liveEventNamingChanged.ts";
import type { LiveEventNodesChanged } from "./liveEventNodesChanged.ts";
import type { LiveEventNodeSettingsChanged } from "./liveEventNodeSettingsChanged.ts";
import type { LiveEventNodesFenced } from "./liveEventNodesFenced.ts";
import type { LiveEventNodesTraffic } from "./liveEventNodesTraffic.ts";
import type { LiveEventNotificationsChanged } from "./liveEventNotificationsChanged.ts";
import type { LiveEventNotificationsSeen } from "./liveEventNotificationsSeen.ts";
import type { LiveEventReachabilityChanged } from "./liveEventReachabilityChanged.ts";
import type { LiveEventReconcilingChanged } from "./liveEventReconcilingChanged.ts";
import type { LiveEventRootsChanged } from "./liveEventRootsChanged.ts";
import type { LiveEventSubtitlesAttempt } from "./liveEventSubtitlesAttempt.ts";
import type { LiveEventSubtitlesChanged } from "./liveEventSubtitlesChanged.ts";
import type { LiveEventSubtitlesProvidersChanged } from "./liveEventSubtitlesProvidersChanged.ts";
import type { LiveEventTargetsChanged } from "./liveEventTargetsChanged.ts";
import type { LiveEventTasksChanged } from "./liveEventTasksChanged.ts";
import type { LiveEventTasksProgress } from "./liveEventTasksProgress.ts";
import type { LiveEventTorrentsChanged } from "./liveEventTorrentsChanged.ts";
import type { LiveEventTorrentsLive } from "./liveEventTorrentsLive.ts";
import type { LiveEventTrackersChanged } from "./liveEventTrackersChanged.ts";
import type { LiveEventTrackersSeedingChanged } from "./liveEventTrackersSeedingChanged.ts";
import type { LiveEventTransfersChanged } from "./liveEventTransfersChanged.ts";
import type { LiveEventVolumesChanged } from "./liveEventVolumesChanged.ts";
import type { LiveEventWorksChanged } from "./liveEventWorksChanged.ts";
import type { LiveEventWorksContentTypesChanged } from "./liveEventWorksContentTypesChanged.ts";
import type { LiveEventWorksRemoved } from "./liveEventWorksRemoved.ts";
import type { LiveEventWorksTypeChanged } from "./liveEventWorksTypeChanged.ts";

export type LiveEvent =
  | LiveEventBackupsChanged
  | LiveEventDownloadClientsChanged
  | LiveEventFilesChanged
  | LiveEventFilesMoved
  | LiveEventFirstrunChanged
  | LiveEventGrabsBlocklistChanged
  | LiveEventGrabsChanged
  | LiveEventGrabsQueueChanged
  | LiveEventGrabsSearchChanged
  | LiveEventHealthChanged
  | LiveEventHistoryChanged
  | LiveEventHomeSettingsChanged
  | LiveEventIdentityAccountsChanged
  | LiveEventIdentityAddressChanged
  | LiveEventIdentityLinksChanged
  | LiveEventIdentityPasskeysChanged
  | LiveEventIdentitySessionsChanged
  | LiveEventImportsChanged
  | LiveEventIndexersCatalogChanged
  | LiveEventIndexersChanged
  | LiveEventIndexersRssChanged
  | LiveEventIndexersSettingsChanged
  | LiveEventIndexersStatusChanged
  | LiveEventLiveAccessChanged
  | LiveEventLiveResync
  | LiveEventLiveSessionEnded
  | LiveEventMediaServersChanged
  | LiveEventMediaServersItemsFound
  | LiveEventMediaServersRecovered
  | LiveEventMediaServersSessionsChanged
  | LiveEventMediaServersUnhealthy
  | LiveEventMetadataChanged
  | LiveEventMovesChanged
  | LiveEventNamingChanged
  | LiveEventNodeSettingsChanged
  | LiveEventNodesChanged
  | LiveEventNodesFenced
  | LiveEventNodesTraffic
  | LiveEventNotificationsChanged
  | LiveEventNotificationsSeen
  | LiveEventReachabilityChanged
  | LiveEventReconcilingChanged
  | LiveEventRootsChanged
  | LiveEventSubtitlesAttempt
  | LiveEventSubtitlesChanged
  | LiveEventSubtitlesProvidersChanged
  | LiveEventTargetsChanged
  | LiveEventTasksChanged
  | LiveEventTasksProgress
  | LiveEventTorrentsChanged
  | LiveEventTorrentsLive
  | LiveEventTrackersChanged
  | LiveEventTrackersSeedingChanged
  | LiveEventTransfersChanged
  | LiveEventVolumesChanged
  | LiveEventWorksChanged
  | LiveEventWorksContentTypesChanged
  | LiveEventWorksRemoved
  | LiveEventWorksTypeChanged;

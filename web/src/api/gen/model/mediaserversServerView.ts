import type { MediaserversKind } from "./mediaserversKind.ts";
import type { MediaserversNodeRef } from "./mediaserversNodeRef.ts";
import type { MediaserversServerViewContact } from "./mediaserversServerViewContact.ts";
import type { MediaserversServerViewFallback } from "./mediaserversServerViewFallback.ts";
import type { MediaserversServerViewSends } from "./mediaserversServerViewSends.ts";
import type { MediaserversState } from "./mediaserversState.ts";
import type { MediaserversUnhealthy } from "./mediaserversUnhealthy.ts";

export interface MediaserversServerView {
  /** What the reaching Node calls, such as http://localhost:8096 */
  address: string;
  /** Where Open in sends a browser, for Jellyfin and Emby; absent for no Open in button */
  browserAddress?: string;
  /** When both routes were last tested */
  checkedAt?: string;
  /** As the reaching Node last reported it; unknown until it reports and while it is offline */
  contact: MediaserversServerViewContact;
  contactSince?: string;
  /** Whether the Controller reaches the server itself, which it does while the reaching Node is offline */
  fallback: MediaserversServerViewFallback;
  /** What the Controller calls; absent when it calls address */
  fallbackAddress?: string;
  id: string;
  /** Plex's machineIdentifier, or the Jellyfin or Emby server Id */
  identity: string;
  kind: MediaserversKind;
  lastContactAt?: string;
  lastSendAt?: string;
  /**
   * Shown roots without a confirmed Path mapping
   * @minimum 0
   */
  mappingsNeeded: number;
  name: string;
  /** Plex only: whether the signed-in account has Plex Pass */
  plexPass?: boolean;
  /** The Node that sends changes and watches sessions */
  reachingNode: MediaserversNodeRef;
  /** The Node the server runs on; absent when it runs on no Node */
  runsOnNode?: MediaserversNodeRef;
  sends: MediaserversServerViewSends;
  state: MediaserversState;
  /** Set while the server is unhealthy */
  unhealthy?: MediaserversUnhealthy;
  /** As the server last reported it */
  version: string;
}

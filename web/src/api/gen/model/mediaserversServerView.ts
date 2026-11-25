import type { MediaserversKind } from "./mediaserversKind.ts";
import type { MediaserversLibraryChoice } from "./mediaserversLibraryChoice.ts";
import type { MediaserversNodeRef } from "./mediaserversNodeRef.ts";
import type { MediaserversServerViewContact } from "./mediaserversServerViewContact.ts";
import type { MediaserversServerViewFallback } from "./mediaserversServerViewFallback.ts";
import type { MediaserversServerViewSends } from "./mediaserversServerViewSends.ts";
import type { MediaserversState } from "./mediaserversState.ts";
import type { MediaserversUnhealthy } from "./mediaserversUnhealthy.ts";

export interface MediaserversServerView {
  /** The address the Connect from Node uses, such as http://localhost:8096 */
  address: string;
  /** The address the Open in button opens, for Jellyfin and Emby; absent when there is no Open in button */
  browserAddress?: string;
  /** When both routes were last tested */
  checkedAt?: string;
  /** As the Connect from Node last reported it; unknown until it reports and while it is offline */
  contact: MediaserversServerViewContact;
  contactSince?: string;
  /** Whether the Controller can connect to the server itself, which it does while the Connect from Node is offline */
  fallback: MediaserversServerViewFallback;
  /** The address the Controller uses while the Connect from Node is offline; absent when it uses address */
  fallbackAddress?: string;
  id: string;
  /** Plex's machineIdentifier, or the Jellyfin or Emby server Id */
  identity: string;
  kind: MediaserversKind;
  lastContactAt?: string;
  lastSendAt?: string;
  /**
   * Roots with shown true and no confirmed Path mapping
   * @minimum 0
   */
  mappingsNeeded: number;
  /** Jellyfin and Emby: the address How to watch gives Members; absent when the Admin set none */
  memberAddress?: string;
  /**
   * How many Members can watch on the server
   * @minimum 0
   */
  memberCount: number;
  /** The Libraries for Members: what a Member's access covers */
  memberLibraries: MediaserversLibraryChoice[];
  /** Show to Members: whether Members may be offered access to the server */
  membersSee: boolean;
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

import type { MediaserversLibraryChoice } from "./mediaserversLibraryChoice.ts";

export interface MediaserversPatchBody {
  /**
   * @minLength 1
   * @maxLength 2048
   */
  address?: string;
  /**
   * Jellyfin and Emby. Write-only. A key from signInToMediaServer keeps the server's saved deviceId, since the key isn't tied to a device
   * @minLength 1
   * @maxLength 1024
   */
  apiKey?: string;
  /**
   * null to hide the Open in button
   * @nullable
   */
  browserAddress?: string | null;
  /**
   * null to use address
   * @nullable
   */
  fallbackAddress?: string | null;
  /**
   * Jellyfin and Emby: the address How to watch gives Members; null to give none
   * @nullable
   */
  memberAddress?: string | null;
  /**
   * The Libraries for Members, from listMediaServerLibraries; never empty while membersSee is on
   * @maxItems 1000
   */
  memberLibraries?: MediaserversLibraryChoice[];
  /** Show to Members. Turning it on the first time picks the libraries behind the shown Path mappings, unless memberLibraries names others */
  membersSee?: boolean;
  /**
   * @minLength 1
   * @maxLength 100
   */
  name?: string;
  /** Pause Conversions on the Node it runs on while it transcodes video */
  pausesConversions?: boolean;
  reachingNodeId?: string;
  /**
   * A Node id; null when the server runs on no Node
   * @nullable
   */
  runsOnNodeId?: string | null;
}

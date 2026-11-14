export interface MediaserversDraftBody {
  /**
   * What the reaching Node calls, such as http://localhost:8096; absent for Plex, whose address SlipStream picks from the server's connections
   * @maxLength 2048
   */
  address?: string;
  /**
   * Jellyfin and Emby. Write-only
   * @maxLength 1024
   */
  apiKey?: string;
  /**
   * Jellyfin and Emby: where Open in sends a browser; absent for no Open in button
   * @maxLength 2048
   */
  browserAddress?: string;
  /**
   * Jellyfin and Emby: the deviceId an administrator sign-in returned
   * @maxLength 256
   */
  deviceId?: string;
  /**
   * What the Controller calls; absent for address
   * @maxLength 2048
   */
  fallbackAddress?: string;
  /**
   * One of the kinds listMediaServerKinds lists
   * @minLength 1
   */
  kind: string;
  /**
   * Such as Living room Plex
   * @minLength 1
   * @maxLength 100
   */
  name: string;
  /**
   * Plex: a machineIdentifier from listPlexServers
   * @maxLength 256
   */
  plexServer?: string;
  /** The Controller, or a Node that holds Media Server link */
  reachingNodeId: string;
  /** The Node the server runs on; absent when it runs on none */
  runsOnNodeId?: string;
}

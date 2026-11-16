export interface MediaserversDraftBody {
  /**
   * The address the Connect from Node uses, such as http://localhost:8096; absent for Plex, whose address SlipStream picks from the server's connections
   * @maxLength 2048
   */
  address?: string;
  /**
   * Jellyfin and Emby. Write-only
   * @maxLength 1024
   */
  apiKey?: string;
  /**
   * Jellyfin and Emby: the address the Open in button opens; absent when there is no Open in button
   * @maxLength 2048
   */
  browserAddress?: string;
  /**
   * Jellyfin and Emby: the deviceId an administrator sign-in returned
   * @maxLength 256
   */
  deviceId?: string;
  /**
   * The address the Controller uses while the Connect from Node is offline; absent when it uses address
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
  /** The Controller, or a Node with the Media Server link Role */
  reachingNodeId: string;
  /** The Node the server runs on; absent when it runs on none */
  runsOnNodeId?: string;
}

export interface MediaserversLibrary {
  /** The library folder the mapping's folder sits inside, as the server names it */
  folder: string;
  /** Plex's section key, or the Jellyfin or Emby library ItemId */
  key: string;
  name: string;
}

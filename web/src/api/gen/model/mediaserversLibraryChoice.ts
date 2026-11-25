export interface MediaserversLibraryChoice {
  /**
   * Plex's section key, or the Jellyfin or Emby library ItemId
   * @minLength 1
   * @maxLength 256
   */
  key: string;
  /**
   * @minLength 1
   * @maxLength 256
   */
  name: string;
}

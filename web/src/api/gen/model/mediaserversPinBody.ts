export interface MediaserversPinBody {
  /**
   * Where plex.tv sends the browser back to once the Admin signs in
   * @maxLength 2048
   */
  forward: string;
}

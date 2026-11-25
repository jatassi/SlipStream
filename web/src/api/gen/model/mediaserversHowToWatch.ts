export interface MediaserversHowToWatch {
  /** The server's Address for Members; absent on Plex, and when the Admin set none */
  address?: string;
  /** The app to get: Plex, Jellyfin or Emby */
  app: string;
  /** The Plex username, or the username on the server */
  signInAs: string;
}

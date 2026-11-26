export interface IdentityServerGrant {
  /** Plex only: true adds the Member to the Plex Home; false shares the server as a friend */
  plexHome?: boolean;
  /** A Media Server shown to Members */
  server: string;
}

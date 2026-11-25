export interface IdentityServerGrant {
  /** Plex only: add the Member to the Plex Home rather than as a friend */
  plexHome?: boolean;
  /** A Media Server shown to Members */
  server: string;
}

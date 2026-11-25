export interface MediaserversPlexPart {
  /** The Admin shared the server in Plex and marked it as added */
  byHand: boolean;
  /** The Member's Plex email; absent until the Member signs in to Plex */
  email?: string;
  /** Offered as, or added to, the Plex Home rather than as a friend */
  plexHome: boolean;
  /** The Member's Plex username; absent until the Member signs in to Plex */
  username?: string;
}

export interface MediaserversPlexPart {
  /** The Admin shared the server in Plex and marked it as added */
  byHand: boolean;
  /** The Member's Plex email; absent until the Member signs in to Plex */
  email?: string;
  /** True when the Member is offered or added to the Plex Home; false for a share as a friend */
  plexHome: boolean;
  /** The Member's Plex username; absent until the Member signs in to Plex */
  username?: string;
}

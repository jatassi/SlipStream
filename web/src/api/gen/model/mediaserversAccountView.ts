export interface MediaserversAccountView {
  /** When the sign-in expires unless renewed */
  expiresAt: string;
  /** The Plex account's name */
  name: string;
  plexPass: boolean;
  /** plex.tv refused the last renewal. Sign in with Plex again. */
  refused: boolean;
  renewedAt?: string;
  signedInAt: string;
}

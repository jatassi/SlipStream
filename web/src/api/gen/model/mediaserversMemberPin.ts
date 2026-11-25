export interface MediaserversMemberPin {
  /** Open it in a new tab, or in this one on a phone, where the Member signs in to Plex */
  authUrl: string;
  expiresAt: string;
  pinId: number;
}

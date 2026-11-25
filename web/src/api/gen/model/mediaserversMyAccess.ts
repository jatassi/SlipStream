import type { MediaserversMyAccessRow } from "./mediaserversMyAccessRow.ts";
import type { MediaserversMyPlex } from "./mediaserversMyPlex.ts";

export interface MediaserversMyAccess {
  /** Whether SlipStream can send email, so the Member hears when a share is made */
  mailSetUp: boolean;
  /** The Plex account the Member signed in to Plex as; absent before a sign-in */
  plex?: MediaserversMyPlex;
  /** Whether How to watch carries Plex's Plex Pass rule: the Member has a Plex row and SlipStream's Plex account has no Plex Pass */
  plexPassLine: boolean;
  /** In the Admin's server order */
  rows: MediaserversMyAccessRow[];
}

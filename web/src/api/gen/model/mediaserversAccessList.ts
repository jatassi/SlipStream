import type { MediaserversAccessServer } from "./mediaserversAccessServer.ts";
import type { MediaserversAccessView } from "./mediaserversAccessView.ts";

export interface MediaserversAccessList {
  /**
   * With account: the servers shown to Members that the Member has no access to yet
   * @nullable
   */
  offerable?: MediaserversAccessServer[] | null;
  /** Whether a Member is in the Plex Home through SlipStream: a shared or active row added to Plex Home */
  plexHomeUsed: boolean;
  /** Whether SlipStream's Plex account has Plex Pass, which Add to Plex Home needs */
  plexPass: boolean;
  /** The rows under Needs you first, then the rest */
  rows: MediaserversAccessView[];
}

import type { MediaserversAccessServerAccess } from "./mediaserversAccessServerAccess.ts";
import type { MediaserversKind } from "./mediaserversKind.ts";

export interface MediaserversAccessServer {
  /** How Members get access: plex_tv through a Plex share, users with a user the Member makes or the Admin links, users_linked with a user the Admin links */
  access: MediaserversAccessServerAccess;
  id: string;
  kind: MediaserversKind;
  name: string;
}

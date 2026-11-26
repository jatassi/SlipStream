import type { MediaserversAccessMember } from "./mediaserversAccessMember.ts";
import type { MediaserversAccessProblem } from "./mediaserversAccessProblem.ts";
import type { MediaserversAccessServer } from "./mediaserversAccessServer.ts";
import type { MediaserversAccessViewActionsItem } from "./mediaserversAccessViewActionsItem.ts";
import type { MediaserversAccessViewState } from "./mediaserversAccessViewState.ts";
import type { MediaserversPlexPart } from "./mediaserversPlexPart.ts";
import type { MediaserversUserPart } from "./mediaserversUserPart.ts";

export interface MediaserversAccessView {
  /** What this Admin may start on the row, in the order the screen shows them */
  actions: MediaserversAccessViewActionsItem[];
  changedAt: string;
  id: string;
  /** The server's libraries for Members by name, for sharing by hand */
  libraries: string[];
  member: MediaserversAccessMember;
  /** Whether the row waits on an Admin: a share to make or accept, or a removal to try again */
  needsYou: boolean;
  /** Plex rows only */
  plex?: MediaserversPlexPart;
  /** The last call that failed on the row; absent once a move succeeds */
  problem?: MediaserversAccessProblem;
  server: MediaserversAccessServer;
  state: MediaserversAccessViewState;
  /** Jellyfin and Emby rows with a user on the server */
  user?: MediaserversUserPart;
}

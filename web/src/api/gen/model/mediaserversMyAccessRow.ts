import type { MediaserversHowToWatch } from "./mediaserversHowToWatch.ts";
import type { MediaserversMyAccessRowActionsItem } from "./mediaserversMyAccessRowActionsItem.ts";
import type { MediaserversMyAccessRowState } from "./mediaserversMyAccessRowState.ts";
import type { MediaserversMyServer } from "./mediaserversMyServer.ts";

export interface MediaserversMyAccessRow {
  /** What the Member may start on the row */
  actions: MediaserversMyAccessRowActionsItem[];
  /** The Admin to ask about this server */
  adminName: string;
  /** Absent while pending */
  howToWatch?: MediaserversHowToWatch;
  id: string;
  /** Whether the Member still has something to do or wait for */
  pending: boolean;
  server: MediaserversMyServer;
  state: MediaserversMyAccessRowState;
}

import type { MediaserversNodeRef } from "./mediaserversNodeRef.ts";
import type { MediaserversPlexConnection } from "./mediaserversPlexConnection.ts";

export interface MediaserversPlexServerView {
  /** Every connection but relays */
  connections: MediaserversPlexConnection[];
  machineIdentifier: string;
  name: string;
  suggestedReaching: MediaserversNodeRef;
  /** The Node whose address a local connection has */
  suggestedRunsOn?: MediaserversNodeRef;
  version: string;
}

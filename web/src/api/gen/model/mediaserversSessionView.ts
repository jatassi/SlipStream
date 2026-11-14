import type { MediaserversSessionViewDecision } from "./mediaserversSessionViewDecision.ts";

export interface MediaserversSessionView {
  decision: MediaserversSessionViewDecision;
  /** The server reports hardware transcoding */
  hardware: boolean;
  /** The file as the server names it */
  path: string;
  playId: string;
  player: string;
  product: string;
  /** Transcode reasons as the server gave them; Plex gives none */
  reasons: string[];
  startedAt: string;
  user: string;
  /** The Work whose file plays; absent when no mapping leads to a library file */
  workId?: string;
}

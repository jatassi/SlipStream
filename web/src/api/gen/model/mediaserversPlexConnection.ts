import type { MediaserversReached } from "./mediaserversReached.ts";

export interface MediaserversPlexConnection {
  https: boolean;
  local: boolean;
  /** The Controller and each Node that holds Media Server link */
  reachedFrom: MediaserversReached[];
  uri: string;
}

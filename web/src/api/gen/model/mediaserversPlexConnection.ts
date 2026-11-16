import type { MediaserversReached } from "./mediaserversReached.ts";

export interface MediaserversPlexConnection {
  https: boolean;
  local: boolean;
  /** The Controller and each Node with the Media Server link Role */
  reachedFrom: MediaserversReached[];
  uri: string;
}

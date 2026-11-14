import type { MediaserversStateLevel } from "./mediaserversStateLevel.ts";

export interface MediaserversState {
  level: MediaserversStateLevel;
  /** The line a person reads, such as Living room Plex hasn't responded to Den since 14:02. */
  words: string;
}

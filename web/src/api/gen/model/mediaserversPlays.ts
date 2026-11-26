import type { MediaserversPlayView } from "./mediaserversPlayView.ts";

export interface MediaserversPlays {
  /** The cursor of the next page; absent on the last */
  next?: string;
  /** Newest first */
  plays: MediaserversPlayView[];
}

import type { MediaserversChangeView } from "./mediaserversChangeView.ts";

export interface MediaserversChanges {
  /** Newest first */
  changes: MediaserversChangeView[];
  /** The cursor of the next page; absent on the last */
  next?: string;
}

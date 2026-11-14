import type { MediaserversKindView } from "./mediaserversKindView.ts";

export interface KindsOutputBody {
  /** In the order the Add sheet offers them */
  kinds: MediaserversKindView[];
}

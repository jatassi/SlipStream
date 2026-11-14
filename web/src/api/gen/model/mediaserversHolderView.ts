import type { MediaserversNodeRef } from "./mediaserversNodeRef.ts";

export interface MediaserversHolderView {
  /** The server's reaching Node now */
  current: boolean;
  fetchedAt: string;
  node: MediaserversNodeRef;
}

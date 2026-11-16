import type { MediaserversNodeRef } from "./mediaserversNodeRef.ts";

export interface MediaserversHolderView {
  /** This Node is the server's Connect from Node now */
  current: boolean;
  fetchedAt: string;
  node: MediaserversNodeRef;
}

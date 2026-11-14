import type { MediaserversHolderView } from "./mediaserversHolderView.ts";

export interface HoldersOutputBody {
  /** Every Node that fetched the server's credential, newest first. A Worker that fetched a Jellyfin or Emby key holds it until the key is replaced on the server */
  holders: MediaserversHolderView[];
}

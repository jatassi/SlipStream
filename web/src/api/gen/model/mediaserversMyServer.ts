import type { MediaserversMyServerAccess } from "./mediaserversMyServerAccess.ts";

export interface MediaserversMyServer {
  access: MediaserversMyServerAccess;
  /** The kind's product name, such as Jellyfin */
  label: string;
  name: string;
}

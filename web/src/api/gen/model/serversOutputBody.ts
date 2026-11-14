import type { MediaserversServerView } from "./mediaserversServerView.ts";

export interface ServersOutputBody {
  /** By name */
  servers: MediaserversServerView[];
}

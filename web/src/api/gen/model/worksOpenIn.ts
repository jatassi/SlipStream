import type { MediaserversKind } from "./mediaserversKind.ts";

export interface WorksOpenIn {
  kind: MediaserversKind;
  /** The Media Server's name, such as Living room Plex */
  name: string;
  serverId: string;
  url: string;
}

import type { MediaserversKind } from "./mediaserversKind.ts";

export interface WorksOpenIn {
  kind: MediaserversKind;
  /** The kind's name, such as Plex */
  label: string;
  /** The Media Server's name, such as Living room Plex */
  name: string;
  serverId: string;
  url: string;
}

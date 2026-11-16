import type { MediaserversLibrary } from "./mediaserversLibrary.ts";
import type { MediaserversNodeRef } from "./mediaserversNodeRef.ts";

export interface MediaserversMappingView {
  /** false while a guess or after the root was re-homed, and changes under it wait */
  confirmed: boolean;
  /** SlipStream pre-filled the folder from shared trailing folder names */
  guess: boolean;
  /** The library the folder sits inside; absent when shown is false */
  library?: MediaserversLibrary;
  rootId: string;
  /** The root's display name, such as Movies library root on Media server */
  rootName: string;
  /** The Storage Node whose disk holds the root */
  rootNode: MediaserversNodeRef;
  /** The server has content under the folder, or a full scan of its library ended */
  scanned: boolean;
  /** The path the server uses for the root's folder; absent when shown is false */
  serverFolder?: string;
  /** false for a root that isn't in one of this server's libraries, which gets no changes */
  shown: boolean;
  /** What to do about the mapping, such as Living room Plex won't get changes for files in Movies (D:). */
  words?: string;
}

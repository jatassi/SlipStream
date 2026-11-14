import type { MediaserversLibrary } from "./mediaserversLibrary.ts";
import type { MediaserversNodeRef } from "./mediaserversNodeRef.ts";

export interface MediaserversMappingView {
  /** false while a guess or after the root was re-homed, and changes under it wait */
  confirmed: boolean;
  /** SlipStream pre-filled the folder from shared trailing folder names */
  guess: boolean;
  /** The library the folder sits inside; absent when not shown */
  library?: MediaserversLibrary;
  rootId: string;
  /** Such as Movies (D:) */
  rootName: string;
  /** The Storage Node whose disk holds the root */
  rootNode: MediaserversNodeRef;
  /** The server has content under the folder, or a full scan of its library ended */
  scanned: boolean;
  /** The root's folder as the server sees it; absent when not shown */
  serverFolder?: string;
  /** false for a root this server doesn't show, which gets no changes */
  shown: boolean;
  /** What to do about the mapping, such as Living room Plex won't get changes for files in Movies (D:). */
  words?: string;
}

import type { DownloadclientsFaultKind } from "./downloadclientsFaultKind.ts";

export interface DownloadclientsFault {
  at: string;
  kind: DownloadclientsFaultKind;
  /** The fault, as a sentence */
  sentence: string;
}

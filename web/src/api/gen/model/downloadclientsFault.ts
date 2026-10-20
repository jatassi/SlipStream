import type { DownloadclientsFaultKind } from "./downloadclientsFaultKind.ts";

export interface DownloadclientsFault {
  at: string;
  kind: DownloadclientsFaultKind;
  /** Why the release looks bad, in plain words */
  sentence: string;
}

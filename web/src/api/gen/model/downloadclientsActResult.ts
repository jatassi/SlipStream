import type { DownloadclientsRefusal } from "./downloadclientsRefusal.ts";

export interface DownloadclientsActResult {
  done: boolean;
  id: string;
  /** Why it wasn't done */
  refusal?: DownloadclientsRefusal;
}

import type { DownloadclientsActResult } from "./downloadclientsActResult.ts";
import type { DownloadclientsPreview } from "./downloadclientsPreview.ts";

export interface DownloadclientsActed {
  /** What a dry run would do */
  preview?: DownloadclientsPreview;
  /** One for each id, in the order sent */
  results: DownloadclientsActResult[];
}

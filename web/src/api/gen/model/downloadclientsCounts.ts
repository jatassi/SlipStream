import type { DownloadclientsNodeCount } from "./downloadclientsNodeCount.ts";
import type { DownloadclientsStateCount } from "./downloadclientsStateCount.ts";
import type { DownloadclientsTabCount } from "./downloadclientsTabCount.ts";

export interface DownloadclientsCounts {
  /** Each Node holding any, in Node order */
  byNode: DownloadclientsNodeCount[];
  /** Each state, in order of precedence */
  byState: DownloadclientsStateCount[];
  /** Each tab, in the view's order */
  byTab: DownloadclientsTabCount[];
}

import type { DownloadclientsDaemon } from "./downloadclientsDaemon.ts";
import type { Problem } from "./problem.ts";
import type { UpdateDownloadClient422Code } from "./updateDownloadClient422Code.ts";

export type UpdateDownloadClient422 = Problem & {
  code?: UpdateDownloadClient422Code;
  /** @nullable */
  daemons?: DownloadclientsDaemon[] | null;
  /** @nullable */
  fields?: string[] | null;
};

import type { DownloadclientsDaemon } from "./downloadclientsDaemon.ts";
import type { Problem } from "./problem.ts";
import type { TestDownloadClient422Code } from "./testDownloadClient422Code.ts";

export type TestDownloadClient422 = Problem & {
  code?: TestDownloadClient422Code;
  /** @nullable */
  daemons?: DownloadclientsDaemon[] | null;
  /** @nullable */
  fields?: string[] | null;
};

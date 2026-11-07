import type { CreateDownloadClient422Code } from "./createDownloadClient422Code.ts";
import type { DownloadclientsDaemon } from "./downloadclientsDaemon.ts";
import type { Problem } from "./problem.ts";

export type CreateDownloadClient422 = Problem & {
  code?: CreateDownloadClient422Code;
  /** @nullable */
  daemons?: DownloadclientsDaemon[] | null;
  /** @nullable */
  fields?: string[] | null;
};

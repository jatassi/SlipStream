import type { Problem } from "./problem.ts";
import type { UpdateDownloadClient422Code } from "./updateDownloadClient422Code.ts";

export type UpdateDownloadClient422 = Problem & {
  code?: UpdateDownloadClient422Code;
  /** @nullable */
  fields?: string[] | null;
};

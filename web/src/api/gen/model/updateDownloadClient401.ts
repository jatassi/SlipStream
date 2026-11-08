import type { Problem } from "./problem.ts";
import type { UpdateDownloadClient401Code } from "./updateDownloadClient401Code.ts";

export type UpdateDownloadClient401 = Problem & {
  code?: UpdateDownloadClient401Code;
};

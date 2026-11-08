import type { Problem } from "./problem.ts";
import type { UpdateDownloadClient502Code } from "./updateDownloadClient502Code.ts";

export type UpdateDownloadClient502 = Problem & {
  code?: UpdateDownloadClient502Code;
};

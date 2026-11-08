import type { Problem } from "./problem.ts";
import type { UpdateDownloadClient404Code } from "./updateDownloadClient404Code.ts";

export type UpdateDownloadClient404 = Problem & {
  code?: UpdateDownloadClient404Code;
};

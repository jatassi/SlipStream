import type { Problem } from "./problem.ts";
import type { UpdateDownloadClient409Code } from "./updateDownloadClient409Code.ts";

export type UpdateDownloadClient409 = Problem & {
  code?: UpdateDownloadClient409Code;
};

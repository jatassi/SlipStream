import type { Problem } from "./problem.ts";
import type { UpdateDownloadClient500Code } from "./updateDownloadClient500Code.ts";

export type UpdateDownloadClient500 = Problem & {
  code?: UpdateDownloadClient500Code;
};

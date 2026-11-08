import type { Problem } from "./problem.ts";
import type { UpdateDownloadClient403Code } from "./updateDownloadClient403Code.ts";

export type UpdateDownloadClient403 = Problem & {
  code?: UpdateDownloadClient403Code;
};

import type { CreateDownloadClient401Code } from "./createDownloadClient401Code.ts";
import type { Problem } from "./problem.ts";

export type CreateDownloadClient401 = Problem & {
  code?: CreateDownloadClient401Code;
};

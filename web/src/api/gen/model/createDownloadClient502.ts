import type { CreateDownloadClient502Code } from "./createDownloadClient502Code.ts";
import type { Problem } from "./problem.ts";

export type CreateDownloadClient502 = Problem & {
  code?: CreateDownloadClient502Code;
};

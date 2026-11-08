import type { CreateDownloadClient500Code } from "./createDownloadClient500Code.ts";
import type { Problem } from "./problem.ts";

export type CreateDownloadClient500 = Problem & {
  code?: CreateDownloadClient500Code;
};

import type { CreateDownloadClient403Code } from "./createDownloadClient403Code.ts";
import type { Problem } from "./problem.ts";

export type CreateDownloadClient403 = Problem & {
  code?: CreateDownloadClient403Code;
};

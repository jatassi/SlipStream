import type { Problem } from "./problem.ts";
import type { UploadRestore409Code } from "./uploadRestore409Code.ts";

export type UploadRestore409 = Problem & {
  code?: UploadRestore409Code;
};

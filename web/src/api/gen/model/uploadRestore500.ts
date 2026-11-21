import type { Problem } from "./problem.ts";
import type { UploadRestore500Code } from "./uploadRestore500Code.ts";

export type UploadRestore500 = Problem & {
  code?: UploadRestore500Code;
};

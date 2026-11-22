import type { Problem } from "./problem.ts";
import type { UploadRestore422Code } from "./uploadRestore422Code.ts";

export type UploadRestore422 = Problem & {
  code?: UploadRestore422Code;
};

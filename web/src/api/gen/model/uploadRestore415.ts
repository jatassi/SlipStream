import type { Problem } from "./problem.ts";
import type { UploadRestore415Code } from "./uploadRestore415Code.ts";

export type UploadRestore415 = Problem & {
  code?: UploadRestore415Code;
};

import type { Problem } from "./problem.ts";
import type { UploadRestore401Code } from "./uploadRestore401Code.ts";

export type UploadRestore401 = Problem & {
  code?: UploadRestore401Code;
};

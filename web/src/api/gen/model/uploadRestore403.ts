import type { Problem } from "./problem.ts";
import type { UploadRestore403Code } from "./uploadRestore403Code.ts";

export type UploadRestore403 = Problem & {
  code?: UploadRestore403Code;
};

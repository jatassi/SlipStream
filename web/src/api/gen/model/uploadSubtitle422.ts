import type { Problem } from "./problem.ts";
import type { UploadSubtitle422Code } from "./uploadSubtitle422Code.ts";

export type UploadSubtitle422 = Problem & {
  code?: UploadSubtitle422Code;
};

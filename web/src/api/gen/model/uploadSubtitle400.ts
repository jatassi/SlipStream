import type { Problem } from "./problem.ts";
import type { UploadSubtitle400Code } from "./uploadSubtitle400Code.ts";

export type UploadSubtitle400 = Problem & {
  code?: UploadSubtitle400Code;
};

import type { Problem } from "./problem.ts";
import type { UploadSubtitle500Code } from "./uploadSubtitle500Code.ts";

export type UploadSubtitle500 = Problem & {
  code?: UploadSubtitle500Code;
};

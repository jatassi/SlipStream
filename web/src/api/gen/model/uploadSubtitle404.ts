import type { Problem } from "./problem.ts";
import type { UploadSubtitle404Code } from "./uploadSubtitle404Code.ts";

export type UploadSubtitle404 = Problem & {
  code?: UploadSubtitle404Code;
};

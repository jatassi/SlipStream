import type { Problem } from "./problem.ts";
import type { UploadSubtitle409Code } from "./uploadSubtitle409Code.ts";

export type UploadSubtitle409 = Problem & {
  code?: UploadSubtitle409Code;
};

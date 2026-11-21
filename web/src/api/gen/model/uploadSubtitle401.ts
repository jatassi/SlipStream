import type { Problem } from "./problem.ts";
import type { UploadSubtitle401Code } from "./uploadSubtitle401Code.ts";

export type UploadSubtitle401 = Problem & {
  code?: UploadSubtitle401Code;
};

import type { GetIndexerCaptcha401Code } from "./getIndexerCaptcha401Code.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerCaptcha401 = Problem & {
  code?: GetIndexerCaptcha401Code;
};

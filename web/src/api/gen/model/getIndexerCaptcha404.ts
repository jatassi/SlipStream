import type { GetIndexerCaptcha404Code } from "./getIndexerCaptcha404Code.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerCaptcha404 = Problem & {
  code?: GetIndexerCaptcha404Code;
};

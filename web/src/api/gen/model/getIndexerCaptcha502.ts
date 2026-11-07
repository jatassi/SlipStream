import type { GetIndexerCaptcha502Code } from "./getIndexerCaptcha502Code.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerCaptcha502 = Problem & {
  code?: GetIndexerCaptcha502Code;
};

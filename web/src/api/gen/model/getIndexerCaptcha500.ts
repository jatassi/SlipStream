import type { GetIndexerCaptcha500Code } from "./getIndexerCaptcha500Code.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerCaptcha500 = Problem & {
  code?: GetIndexerCaptcha500Code;
};

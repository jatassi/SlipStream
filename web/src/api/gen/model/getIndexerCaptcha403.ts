import type { GetIndexerCaptcha403Code } from "./getIndexerCaptcha403Code.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerCaptcha403 = Problem & {
  code?: GetIndexerCaptcha403Code;
};

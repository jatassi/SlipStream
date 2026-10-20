import type { Problem } from "./problem.ts";
import type { RefreshContentType422Code } from "./refreshContentType422Code.ts";

export type RefreshContentType422 = Problem & {
  code?: RefreshContentType422Code;
};

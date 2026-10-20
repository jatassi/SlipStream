import type { Problem } from "./problem.ts";
import type { RefreshContentType409Code } from "./refreshContentType409Code.ts";

export type RefreshContentType409 = Problem & {
  code?: RefreshContentType409Code;
};

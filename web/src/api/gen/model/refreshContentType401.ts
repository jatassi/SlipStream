import type { Problem } from "./problem.ts";
import type { RefreshContentType401Code } from "./refreshContentType401Code.ts";

export type RefreshContentType401 = Problem & {
  code?: RefreshContentType401Code;
};

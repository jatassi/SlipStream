import type { Problem } from "./problem.ts";
import type { RefreshContentType500Code } from "./refreshContentType500Code.ts";

export type RefreshContentType500 = Problem & {
  code?: RefreshContentType500Code;
};

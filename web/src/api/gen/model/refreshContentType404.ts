import type { Problem } from "./problem.ts";
import type { RefreshContentType404Code } from "./refreshContentType404Code.ts";

export type RefreshContentType404 = Problem & {
  code?: RefreshContentType404Code;
};

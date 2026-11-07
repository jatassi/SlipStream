import type { Problem } from "./problem.ts";
import type { RefreshWorks422Code } from "./refreshWorks422Code.ts";

export type RefreshWorks422 = Problem & {
  code?: RefreshWorks422Code;
};

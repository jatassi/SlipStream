import type { Problem } from "./problem.ts";
import type { RefreshWorks409Code } from "./refreshWorks409Code.ts";

export type RefreshWorks409 = Problem & {
  code?: RefreshWorks409Code;
};

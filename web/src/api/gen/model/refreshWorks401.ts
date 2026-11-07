import type { Problem } from "./problem.ts";
import type { RefreshWorks401Code } from "./refreshWorks401Code.ts";

export type RefreshWorks401 = Problem & {
  code?: RefreshWorks401Code;
};

import type { Problem } from "./problem.ts";
import type { RefreshWorks500Code } from "./refreshWorks500Code.ts";

export type RefreshWorks500 = Problem & {
  code?: RefreshWorks500Code;
};

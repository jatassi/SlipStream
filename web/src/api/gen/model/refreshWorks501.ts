import type { Problem } from "./problem.ts";
import type { RefreshWorks501Code } from "./refreshWorks501Code.ts";

export type RefreshWorks501 = Problem & {
  code?: RefreshWorks501Code;
};

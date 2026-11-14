import type { Problem } from "./problem.ts";
import type { RefreshMediaServer500Code } from "./refreshMediaServer500Code.ts";

export type RefreshMediaServer500 = Problem & {
  code?: RefreshMediaServer500Code;
};

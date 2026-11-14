import type { Problem } from "./problem.ts";
import type { RefreshMediaServer401Code } from "./refreshMediaServer401Code.ts";

export type RefreshMediaServer401 = Problem & {
  code?: RefreshMediaServer401Code;
};

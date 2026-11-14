import type { Problem } from "./problem.ts";
import type { RefreshMediaServer404Code } from "./refreshMediaServer404Code.ts";

export type RefreshMediaServer404 = Problem & {
  code?: RefreshMediaServer404Code;
};

import type { Problem } from "./problem.ts";
import type { RefreshMediaServer422Code } from "./refreshMediaServer422Code.ts";

export type RefreshMediaServer422 = Problem & {
  code?: RefreshMediaServer422Code;
};

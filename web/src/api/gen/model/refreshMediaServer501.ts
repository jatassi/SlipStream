import type { Problem } from "./problem.ts";
import type { RefreshMediaServer501Code } from "./refreshMediaServer501Code.ts";

export type RefreshMediaServer501 = Problem & {
  code?: RefreshMediaServer501Code;
};

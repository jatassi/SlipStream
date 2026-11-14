import type { Problem } from "./problem.ts";
import type { RefreshMediaServer409Code } from "./refreshMediaServer409Code.ts";

export type RefreshMediaServer409 = Problem & {
  code?: RefreshMediaServer409Code;
};

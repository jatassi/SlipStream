import type { Problem } from "./problem.ts";
import type { RefreshWork409Code } from "./refreshWork409Code.ts";

export type RefreshWork409 = Problem & {
  code?: RefreshWork409Code;
};

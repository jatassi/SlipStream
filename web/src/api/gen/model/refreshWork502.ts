import type { Problem } from "./problem.ts";
import type { RefreshWork502Code } from "./refreshWork502Code.ts";

export type RefreshWork502 = Problem & {
  code?: RefreshWork502Code;
};

import type { Problem } from "./problem.ts";
import type { RefreshWork401Code } from "./refreshWork401Code.ts";

export type RefreshWork401 = Problem & {
  code?: RefreshWork401Code;
};

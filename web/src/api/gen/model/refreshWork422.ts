import type { Problem } from "./problem.ts";
import type { RefreshWork422Code } from "./refreshWork422Code.ts";

export type RefreshWork422 = Problem & {
  code?: RefreshWork422Code;
};

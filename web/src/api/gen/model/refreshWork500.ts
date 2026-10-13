import type { Problem } from "./problem.ts";
import type { RefreshWork500Code } from "./refreshWork500Code.ts";

export type RefreshWork500 = Problem & {
  code?: RefreshWork500Code;
};

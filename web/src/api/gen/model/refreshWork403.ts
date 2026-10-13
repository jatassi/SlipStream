import type { Problem } from "./problem.ts";
import type { RefreshWork403Code } from "./refreshWork403Code.ts";

export type RefreshWork403 = Problem & {
  code?: RefreshWork403Code;
};

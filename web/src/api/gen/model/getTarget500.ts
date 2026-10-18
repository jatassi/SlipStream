import type { GetTarget500Code } from "./getTarget500Code.ts";
import type { Problem } from "./problem.ts";

export type GetTarget500 = Problem & {
  code?: GetTarget500Code;
};

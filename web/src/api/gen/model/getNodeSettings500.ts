import type { GetNodeSettings500Code } from "./getNodeSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeSettings500 = Problem & {
  code?: GetNodeSettings500Code;
};

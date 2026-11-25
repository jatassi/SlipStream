import type { GetNodeNetwork500Code } from "./getNodeNetwork500Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeNetwork500 = Problem & {
  code?: GetNodeNetwork500Code;
};

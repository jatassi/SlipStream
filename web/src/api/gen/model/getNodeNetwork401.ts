import type { GetNodeNetwork401Code } from "./getNodeNetwork401Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeNetwork401 = Problem & {
  code?: GetNodeNetwork401Code;
};

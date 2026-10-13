import type { GetNodeSettings401Code } from "./getNodeSettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeSettings401 = Problem & {
  code?: GetNodeSettings401Code;
};

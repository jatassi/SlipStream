import type { GetNodeConfig401Code } from "./getNodeConfig401Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeConfig401 = Problem & {
  code?: GetNodeConfig401Code;
};

import type { GetNodeConfig503Code } from "./getNodeConfig503Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeConfig503 = Problem & {
  code?: GetNodeConfig503Code;
};

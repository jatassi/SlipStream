import type { Problem } from "./problem.ts";
import type { UpdateNodeConfig503Code } from "./updateNodeConfig503Code.ts";

export type UpdateNodeConfig503 = Problem & {
  code?: UpdateNodeConfig503Code;
};

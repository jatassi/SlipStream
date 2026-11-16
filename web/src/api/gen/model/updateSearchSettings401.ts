import type { Problem } from "./problem.ts";
import type { UpdateSearchSettings401Code } from "./updateSearchSettings401Code.ts";

export type UpdateSearchSettings401 = Problem & {
  code?: UpdateSearchSettings401Code;
};

import type { Problem } from "./problem.ts";
import type { UpdateSearchSettings500Code } from "./updateSearchSettings500Code.ts";

export type UpdateSearchSettings500 = Problem & {
  code?: UpdateSearchSettings500Code;
};

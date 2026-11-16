import type { Problem } from "./problem.ts";
import type { UpdateSearchSettings403Code } from "./updateSearchSettings403Code.ts";

export type UpdateSearchSettings403 = Problem & {
  code?: UpdateSearchSettings403Code;
};

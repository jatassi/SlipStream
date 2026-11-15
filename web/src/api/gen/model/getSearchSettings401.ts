import type { GetSearchSettings401Code } from "./getSearchSettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetSearchSettings401 = Problem & {
  code?: GetSearchSettings401Code;
};

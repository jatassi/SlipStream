import type { GetSearchSettings500Code } from "./getSearchSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetSearchSettings500 = Problem & {
  code?: GetSearchSettings500Code;
};

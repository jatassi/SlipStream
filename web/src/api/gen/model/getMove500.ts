import type { GetMove500Code } from "./getMove500Code.ts";
import type { Problem } from "./problem.ts";

export type GetMove500 = Problem & {
  code?: GetMove500Code;
};

import type { GetSession500Code } from "./getSession500Code.ts";
import type { Problem } from "./problem.ts";

export type GetSession500 = Problem & {
  code?: GetSession500Code;
};

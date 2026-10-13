import type { GetNode401Code } from "./getNode401Code.ts";
import type { Problem } from "./problem.ts";

export type GetNode401 = Problem & {
  code?: GetNode401Code;
};

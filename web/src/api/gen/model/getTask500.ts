import type { GetTask500Code } from "./getTask500Code.ts";
import type { Problem } from "./problem.ts";

export type GetTask500 = Problem & {
  code?: GetTask500Code;
};

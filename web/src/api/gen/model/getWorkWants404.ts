import type { GetWorkWants404Code } from "./getWorkWants404Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkWants404 = Problem & {
  code?: GetWorkWants404Code;
};

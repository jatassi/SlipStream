import type { GetTask409Code } from "./getTask409Code.ts";
import type { Problem } from "./problem.ts";

export type GetTask409 = Problem & {
  code?: GetTask409Code;
};

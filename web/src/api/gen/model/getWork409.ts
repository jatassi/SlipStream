import type { GetWork409Code } from "./getWork409Code.ts";
import type { Problem } from "./problem.ts";

export type GetWork409 = Problem & {
  code?: GetWork409Code;
};

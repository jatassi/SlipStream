import type { GetWork404Code } from "./getWork404Code.ts";
import type { Problem } from "./problem.ts";

export type GetWork404 = Problem & {
  code?: GetWork404Code;
};

import type { AddWork404Code } from "./addWork404Code.ts";
import type { Problem } from "./problem.ts";

export type AddWork404 = Problem & {
  code?: AddWork404Code;
};

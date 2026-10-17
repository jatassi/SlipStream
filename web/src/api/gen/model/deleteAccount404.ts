import type { DeleteAccount404Code } from "./deleteAccount404Code.ts";
import type { Problem } from "./problem.ts";

export type DeleteAccount404 = Problem & {
  code?: DeleteAccount404Code;
};

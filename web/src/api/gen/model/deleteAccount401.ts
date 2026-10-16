import type { DeleteAccount401Code } from "./deleteAccount401Code.ts";
import type { Problem } from "./problem.ts";

export type DeleteAccount401 = Problem & {
  code?: DeleteAccount401Code;
};

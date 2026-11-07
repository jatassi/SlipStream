import type { ListLibrary401Code } from "./listLibrary401Code.ts";
import type { Problem } from "./problem.ts";

export type ListLibrary401 = Problem & {
  code?: ListLibrary401Code;
};

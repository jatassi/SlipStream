import type { ListLibrary500Code } from "./listLibrary500Code.ts";
import type { Problem } from "./problem.ts";

export type ListLibrary500 = Problem & {
  code?: ListLibrary500Code;
};

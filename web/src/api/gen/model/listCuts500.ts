import type { ListCuts500Code } from "./listCuts500Code.ts";
import type { Problem } from "./problem.ts";

export type ListCuts500 = Problem & {
  code?: ListCuts500Code;
};

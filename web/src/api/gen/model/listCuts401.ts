import type { ListCuts401Code } from "./listCuts401Code.ts";
import type { Problem } from "./problem.ts";

export type ListCuts401 = Problem & {
  code?: ListCuts401Code;
};

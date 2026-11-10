import type { ListNaming500Code } from "./listNaming500Code.ts";
import type { Problem } from "./problem.ts";

export type ListNaming500 = Problem & {
  code?: ListNaming500Code;
};

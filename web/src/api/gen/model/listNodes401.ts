import type { ListNodes401Code } from "./listNodes401Code.ts";
import type { Problem } from "./problem.ts";

export type ListNodes401 = Problem & {
  code?: ListNodes401Code;
};

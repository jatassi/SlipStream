import type { ListNodes500Code } from "./listNodes500Code.ts";
import type { Problem } from "./problem.ts";

export type ListNodes500 = Problem & {
  code?: ListNodes500Code;
};

import type { ListHistory500Code } from "./listHistory500Code.ts";
import type { Problem } from "./problem.ts";

export type ListHistory500 = Problem & {
  code?: ListHistory500Code;
};

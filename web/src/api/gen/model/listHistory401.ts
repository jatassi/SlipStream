import type { ListHistory401Code } from "./listHistory401Code.ts";
import type { Problem } from "./problem.ts";

export type ListHistory401 = Problem & {
  code?: ListHistory401Code;
};

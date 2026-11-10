import type { Problem } from "./problem.ts";
import type { RemoveFromBlocklist401Code } from "./removeFromBlocklist401Code.ts";

export type RemoveFromBlocklist401 = Problem & {
  code?: RemoveFromBlocklist401Code;
};

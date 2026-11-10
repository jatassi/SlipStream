import type { Problem } from "./problem.ts";
import type { RemoveFromBlocklist500Code } from "./removeFromBlocklist500Code.ts";

export type RemoveFromBlocklist500 = Problem & {
  code?: RemoveFromBlocklist500Code;
};

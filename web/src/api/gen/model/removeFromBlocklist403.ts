import type { Problem } from "./problem.ts";
import type { RemoveFromBlocklist403Code } from "./removeFromBlocklist403Code.ts";

export type RemoveFromBlocklist403 = Problem & {
  code?: RemoveFromBlocklist403Code;
};

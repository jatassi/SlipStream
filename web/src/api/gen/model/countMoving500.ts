import type { CountMoving500Code } from "./countMoving500Code.ts";
import type { Problem } from "./problem.ts";

export type CountMoving500 = Problem & {
  code?: CountMoving500Code;
};

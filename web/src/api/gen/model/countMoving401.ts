import type { CountMoving401Code } from "./countMoving401Code.ts";
import type { Problem } from "./problem.ts";

export type CountMoving401 = Problem & {
  code?: CountMoving401Code;
};

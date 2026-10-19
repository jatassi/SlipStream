import type { Problem } from "./problem.ts";
import type { SetWorkRoot403Code } from "./setWorkRoot403Code.ts";

export type SetWorkRoot403 = Problem & {
  code?: SetWorkRoot403Code;
};

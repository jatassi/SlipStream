import type { Problem } from "./problem.ts";
import type { SetWorkOptions403Code } from "./setWorkOptions403Code.ts";

export type SetWorkOptions403 = Problem & {
  code?: SetWorkOptions403Code;
};

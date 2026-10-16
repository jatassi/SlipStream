import type { Problem } from "./problem.ts";
import type { SetAddress403Code } from "./setAddress403Code.ts";

export type SetAddress403 = Problem & {
  code?: SetAddress403Code;
};

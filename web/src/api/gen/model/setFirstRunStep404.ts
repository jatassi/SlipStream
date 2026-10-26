import type { Problem } from "./problem.ts";
import type { SetFirstRunStep404Code } from "./setFirstRunStep404Code.ts";

export type SetFirstRunStep404 = Problem & {
  code?: SetFirstRunStep404Code;
};

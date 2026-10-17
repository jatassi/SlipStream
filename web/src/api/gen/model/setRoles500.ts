import type { Problem } from "./problem.ts";
import type { SetRoles500Code } from "./setRoles500Code.ts";

export type SetRoles500 = Problem & {
  code?: SetRoles500Code;
};

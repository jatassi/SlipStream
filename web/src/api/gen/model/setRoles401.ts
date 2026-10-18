import type { Problem } from "./problem.ts";
import type { SetRoles401Code } from "./setRoles401Code.ts";

export type SetRoles401 = Problem & {
  code?: SetRoles401Code;
};

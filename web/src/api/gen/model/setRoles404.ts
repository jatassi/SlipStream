import type { Problem } from "./problem.ts";
import type { SetRoles404Code } from "./setRoles404Code.ts";

export type SetRoles404 = Problem & {
  code?: SetRoles404Code;
};

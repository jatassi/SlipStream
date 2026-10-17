import type { Problem } from "./problem.ts";
import type { SetRoles403Code } from "./setRoles403Code.ts";

export type SetRoles403 = Problem & {
  code?: SetRoles403Code;
};

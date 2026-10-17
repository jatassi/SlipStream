import type { GetRoleBoard401Code } from "./getRoleBoard401Code.ts";
import type { Problem } from "./problem.ts";

export type GetRoleBoard401 = Problem & {
  code?: GetRoleBoard401Code;
};

import type { CreateJoinCode401Code } from "./createJoinCode401Code.ts";
import type { Problem } from "./problem.ts";

export type CreateJoinCode401 = Problem & {
  code?: CreateJoinCode401Code;
};

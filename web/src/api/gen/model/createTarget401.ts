import type { CreateTarget401Code } from "./createTarget401Code.ts";
import type { Problem } from "./problem.ts";

export type CreateTarget401 = Problem & {
  code?: CreateTarget401Code;
};

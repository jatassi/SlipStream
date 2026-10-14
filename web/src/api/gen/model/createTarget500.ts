import type { CreateTarget500Code } from "./createTarget500Code.ts";
import type { Problem } from "./problem.ts";

export type CreateTarget500 = Problem & {
  code?: CreateTarget500Code;
};

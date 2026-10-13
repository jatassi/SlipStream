import type { GetHealth500Code } from "./getHealth500Code.ts";
import type { Problem } from "./problem.ts";

export type GetHealth500 = Problem & {
  code?: GetHealth500Code;
};

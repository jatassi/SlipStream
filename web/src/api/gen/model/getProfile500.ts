import type { GetProfile500Code } from "./getProfile500Code.ts";
import type { Problem } from "./problem.ts";

export type GetProfile500 = Problem & {
  code?: GetProfile500Code;
};

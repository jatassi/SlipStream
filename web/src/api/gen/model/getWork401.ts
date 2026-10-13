import type { GetWork401Code } from "./getWork401Code.ts";
import type { Problem } from "./problem.ts";

export type GetWork401 = Problem & {
  code?: GetWork401Code;
};

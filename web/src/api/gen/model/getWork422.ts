import type { GetWork422Code } from "./getWork422Code.ts";
import type { Problem } from "./problem.ts";

export type GetWork422 = Problem & {
  code?: GetWork422Code;
};

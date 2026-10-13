import type { GetWork500Code } from "./getWork500Code.ts";
import type { Problem } from "./problem.ts";

export type GetWork500 = Problem & {
  code?: GetWork500Code;
};

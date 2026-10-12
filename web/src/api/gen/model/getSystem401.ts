import type { GetSystem401Code } from "./getSystem401Code.ts";
import type { Problem } from "./problem.ts";

export type GetSystem401 = Problem & {
  code?: GetSystem401Code;
};

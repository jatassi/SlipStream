import type { GetRootSettings500Code } from "./getRootSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetRootSettings500 = Problem & {
  code?: GetRootSettings500Code;
};

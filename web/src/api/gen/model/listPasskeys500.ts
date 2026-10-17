import type { ListPasskeys500Code } from "./listPasskeys500Code.ts";
import type { Problem } from "./problem.ts";

export type ListPasskeys500 = Problem & {
  code?: ListPasskeys500Code;
};

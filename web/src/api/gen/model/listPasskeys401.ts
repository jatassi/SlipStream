import type { ListPasskeys401Code } from "./listPasskeys401Code.ts";
import type { Problem } from "./problem.ts";

export type ListPasskeys401 = Problem & {
  code?: ListPasskeys401Code;
};

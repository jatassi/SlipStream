import type { ListPasskeys403Code } from "./listPasskeys403Code.ts";
import type { Problem } from "./problem.ts";

export type ListPasskeys403 = Problem & {
  code?: ListPasskeys403Code;
};

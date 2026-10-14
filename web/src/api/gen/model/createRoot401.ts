import type { CreateRoot401Code } from "./createRoot401Code.ts";
import type { Problem } from "./problem.ts";

export type CreateRoot401 = Problem & {
  code?: CreateRoot401Code;
};

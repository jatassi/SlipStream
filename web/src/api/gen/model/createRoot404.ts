import type { CreateRoot404Code } from "./createRoot404Code.ts";
import type { Problem } from "./problem.ts";

export type CreateRoot404 = Problem & {
  code?: CreateRoot404Code;
};

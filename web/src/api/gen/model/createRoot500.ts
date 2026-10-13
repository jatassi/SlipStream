import type { CreateRoot500Code } from "./createRoot500Code.ts";
import type { Problem } from "./problem.ts";

export type CreateRoot500 = Problem & {
  code?: CreateRoot500Code;
};

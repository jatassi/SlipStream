import type { CreateRoot403Code } from "./createRoot403Code.ts";
import type { Problem } from "./problem.ts";

export type CreateRoot403 = Problem & {
  code?: CreateRoot403Code;
};

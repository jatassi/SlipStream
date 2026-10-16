import type { Problem } from "./problem.ts";
import type { SetAddress422Code } from "./setAddress422Code.ts";

export type SetAddress422 = Problem & {
  code?: SetAddress422Code;
};

import type { GetProfile404Code } from "./getProfile404Code.ts";
import type { Problem } from "./problem.ts";

export type GetProfile404 = Problem & {
  code?: GetProfile404Code;
};

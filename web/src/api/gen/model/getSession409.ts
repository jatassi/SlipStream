import type { GetSession409Code } from "./getSession409Code.ts";
import type { Problem } from "./problem.ts";

export type GetSession409 = Problem & {
  code?: GetSession409Code;
};

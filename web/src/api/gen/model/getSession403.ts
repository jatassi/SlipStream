import type { GetSession403Code } from "./getSession403Code.ts";
import type { Problem } from "./problem.ts";

export type GetSession403 = Problem & {
  code?: GetSession403Code;
};

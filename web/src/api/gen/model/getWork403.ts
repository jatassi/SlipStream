import type { GetWork403Code } from "./getWork403Code.ts";
import type { Problem } from "./problem.ts";

export type GetWork403 = Problem & {
  code?: GetWork403Code;
};
